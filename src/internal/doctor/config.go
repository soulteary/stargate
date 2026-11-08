// Package doctor provides scoped, redacted deployment diagnostics without application initialization.
package doctor

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
)

const maxConfigBytes = 1 << 20

// Manifest holds only explicitly supplied service configuration. Values are never rendered.
type Manifest struct {
	GatewayURL    string            `json:"gateway_url,omitempty"`
	GatewayCAFile string            `json:"gateway_tls_ca_file,omitempty"`
	Gateway       map[string]string `json:"gateway"`
	Warden        map[string]string `json:"warden,omitempty"`
	Herald        map[string]string `json:"herald,omitempty"`
	TOTP          map[string]string `json:"totp,omitempty"`
}

var gatewayKeys = strings.Fields(`AUTH_HOST COOKIE_DOMAIN COOKIE_SECURE CALLBACK_ALLOWED_HOSTS SESSION_EXCHANGE_SECRET SESSION_EXCHANGE_REDIRECT_PATH PASSWORDS WARDEN_ENABLED WARDEN_URL WARDEN_API_KEY WARDEN_HMAC_KEY_ID WARDEN_HMAC_SECRET WARDEN_TLS_CA_CERT_FILE WARDEN_TLS_CLIENT_CERT_FILE WARDEN_TLS_CLIENT_KEY_FILE WARDEN_TLS_SERVER_NAME HERALD_ENABLED HERALD_URL HERALD_API_KEY HERALD_HMAC_KEY_ID HERALD_HMAC_SECRET HERALD_TLS_CA_CERT_FILE HERALD_TLS_CLIENT_CERT_FILE HERALD_TLS_CLIENT_KEY_FILE HERALD_TLS_SERVER_NAME HERALD_TOTP_ENABLED SESSION_STORAGE_ENABLED SESSION_STORAGE_REDIS_ADDR`)

// CurrentEnvironment snapshots documented gateway fields only. It never initializes clients.
func CurrentEnvironment(lookup func(string) (string, bool)) Manifest {
	m := Manifest{Gateway: map[string]string{}}
	for _, key := range gatewayKeys {
		if value, ok := lookup(key); ok {
			m.Gateway[key] = value
		}
	}
	return m
}

// ReadManifest reads bounded JSON, rejecting duplicates and unknown service names.
func ReadManifest(r io.Reader) (Manifest, error) {
	var m Manifest
	data, err := io.ReadAll(io.LimitReader(r, maxConfigBytes+1))
	if err != nil || len(data) > maxConfigBytes {
		return m, errors.New("cannot read bounded configuration")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err := rejectDuplicates(decoder); err != nil {
		return m, errors.New("invalid configuration JSON")
	}
	if _, err := decoder.Token(); err != io.EOF {
		return m, errors.New("configuration must contain one JSON object")
	}
	decoder = json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil || m.Gateway == nil {
		return Manifest{}, errors.New("configuration requires a gateway object and string values")
	}
	return m, nil
}

func rejectDuplicates(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		if _, stringValue := token.(string); !stringValue {
			return errors.New("string values required")
		}
		return nil
	}
	if delimiter != '{' {
		return errors.New("objects only")
	}
	seen := map[string]bool{}
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return errors.New("duplicate key")
		}
		seen[key] = true
		if err := rejectDuplicates(d); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}

// LoadFile never includes the file name or underlying OS error in diagnostics.
func LoadFile(path string) (Manifest, error) {
	initial, statErr := os.Stat(path)
	if statErr != nil || !initial.Mode().IsRegular() {
		return Manifest{}, errors.New("configuration must be a readable regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, errors.New("cannot open configuration file")
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return Manifest{}, errors.New("configuration must be a regular file")
	}
	m, err := ReadManifest(f)
	if err != nil {
		return m, err
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 && hasSecrets(m) {
		return Manifest{}, errors.New("credential-bearing configuration must not be readable by group or others")
	}
	return m, nil
}

func hasSecrets(m Manifest) bool {
	for _, service := range []map[string]string{m.Gateway, m.Warden, m.Herald, m.TOTP} {
		for key, value := range service {
			if value != "" && (strings.Contains(key, "SECRET") || strings.Contains(key, "KEY") || strings.Contains(key, "PASSWORD")) {
				return true
			}
		}
	}
	return false
}
