package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func fixture() Manifest {
	return Manifest{Gateway: map[string]string{"AUTH_HOST": "auth.example.test", "SESSION_EXCHANGE_SECRET": "synthetic-exchange-secret-123456789", "COOKIE_SECURE": "true", "WARDEN_ENABLED": "true", "WARDEN_URL": "https://directory.example.test", "WARDEN_HMAC_KEY_ID": "gateway", "WARDEN_HMAC_SECRET": "synthetic-warden-secret-123456789"}, Warden: map[string]string{"WARDEN_HMAC_KEYS": `{"gateway":"synthetic-warden-secret-123456789"}`}}
}
func find(r Report, id string) Check {
	for _, c := range r.Checks {
		if c.ID == id {
			return c
		}
	}
	return Check{}
}

func TestOfflinePairingAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		name       string
		change     func(*Manifest)
		id, status string
	}{
		{"matching", func(*Manifest) {}, "warden.credentials", "pass"},
		{"unknown server", func(m *Manifest) { m.Warden = nil }, "warden.credentials", "unknown"},
		{"wrong identifier", func(m *Manifest) { m.Gateway["WARDEN_HMAC_KEY_ID"] = "secret-in-identifier" }, "warden.credentials", "fail"},
		{"wrong secret", func(m *Manifest) { m.Gateway["WARDEN_HMAC_SECRET"] = "secret-in-value" }, "warden.credentials", "fail"},
		{"secure cookie over public HTTP", func(m *Manifest) { m.GatewayURL = "http://auth.example.test" }, "gateway.cookie_transport", "fail"},
		{"cookie parent", func(m *Manifest) { m.Gateway["COOKIE_DOMAIN"] = "unrelated.example.test" }, "gateway.host_cookie", "fail"},
		{"callback URL", func(m *Manifest) {
			m.Gateway["CALLBACK_ALLOWED_HOSTS"] = "https://secret-in-url.test/?token=secret-in-value"
		}, "gateway.callbacks", "fail"},
		{"empty service URL", func(m *Manifest) { m.Gateway["WARDEN_URL"] = " " }, "warden.url", "fail"},
		{"service credentials in URL", func(m *Manifest) { m.Gateway["WARDEN_URL"] = "https://user:secret-in-value@directory.example.test" }, "warden.url", "fail"},
		{"bad password recipe", func(m *Manifest) { m.Gateway["PASSWORDS"] = "secret-in-value" }, "gateway.password_config", "fail"},
		{"bad boolean", func(m *Manifest) { m.Gateway["COOKIE_SECURE"] = "secret-in-value" }, "gateway.cookie_secure", "fail"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := fixture()
			tc.change(&m)
			r := Inspect(m)
			if c := find(r, tc.id); c.Status != tc.status {
				t.Fatalf("check %s status=%s", tc.id, c.Status)
			}
			data, _ := json.Marshal(r)
			for _, secret := range []string{"synthetic-warden-secret", "secret-in-identifier", "secret-in-value", "secret-in-url"} {
				if bytes.Contains(data, []byte(secret)) {
					t.Fatal("diagnostic exposed an input value")
				}
			}
		})
	}
}

func TestHeraldAndTOTPSelectionContracts(t *testing.T) {
	client := map[string]string{"HERALD_HMAC_SECRET": "synthetic-secret", "HERALD_HMAC_KEY_ID": "gateway", "HERALD_API_KEY": "api"}
	server := map[string]string{"REQUEST_AUTH_MODE": "api_key", "API_KEY": "api"}
	r := Report{}
	inspectPair(&r, "pair", client, server, "HERALD", true)
	if r.Checks[0].Status != "fail" {
		t.Fatal("mixed client did not select HMAC")
	}
	server = map[string]string{"HERALD_HMAC_SECRET": "synthetic-secret"}
	r = Report{}
	inspectPair(&r, "pair", client, server, "HERALD", true)
	if r.Checks[0].Status != "fail" {
		t.Fatal("single-secret Herald accepted arbitrary key identity")
	}
	client["HERALD_HMAC_KEY_ID"] = "default"
	r = Report{}
	inspectPair(&r, "pair", client, server, "HERALD", true)
	if r.Checks[0].Status != "pass" {
		t.Fatal("single-secret compatibility lost")
	}
	proxy := map[string]string{"HERALD_TOTP_HMAC_SECRET": "synthetic-secret"}
	totp := map[string]string{"HERALD_TOTP_HMAC_KEYS": `{"sole":"synthetic-secret"}`}
	r = Report{}
	inspectPair(&r, "pair", proxy, totp, "HERALD_TOTP", false)
	if r.Checks[0].Status != "pass" {
		t.Fatal("published singleton-map compatibility lost")
	}
}

func TestManifestRejectsMalformedInputsWithoutValues(t *testing.T) {
	for _, raw := range []string{`{"gateway":{"AUTH_HOST":"a","AUTH_HOST":"secret"}}`, `{"gateway":{},"gateway":{"x":"secret"}}`, `{"gateway":{"x":1}}`, `{"gateway":{"COOKIE_DOMAIN":null}}`, `{"gateway":{"COOKIE_SECURE":false}}`, `{"gateway":{},"unknown":"secret"}`, `{"gateway":{}} {}`, `[]`, strings.Repeat("x", maxConfigBytes+1)} {
		if _, err := ReadManifest(strings.NewReader(raw)); err == nil {
			t.Fatal("invalid manifest accepted")
		} else if strings.Contains(err.Error(), "secret") {
			t.Fatal("input echoed")
		}
	}
	if _, err := ReadManifest(strings.NewReader(`{"gateway":{"AUTH_HOST":"auth.example.test"}}`)); err != nil {
		t.Fatal(err)
	}
}

func TestCommandOfflineNeverUsesNetworkAndDoesNotEchoBadArguments(t *testing.T) {
	var out, errOut bytes.Buffer
	m := fixture()
	data, _ := json.Marshal(m)
	file := t.TempDir() + "/config.json"
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	if code := Run([]string{"--config", file, "--json"}, func(string) (string, bool) { panic("file mode must not inspect environment") }, &out, &errOut); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	for _, args := range [][]string{{"--secret-in-argument"}, {"secret-in-argument"}, {"--config", "secret-in-path"}} {
		out.Reset()
		errOut.Reset()
		if code := Run(args, func(string) (string, bool) { return "", false }, &out, &errOut); code != 2 {
			t.Fatal("bad input did not return2")
		}
		if strings.Contains(out.String()+errOut.String(), "secret-in-") {
			t.Fatal("CLI echoed sensitive argument")
		}
	}
}

func TestHealthBoundaries(t *testing.T) {
	hits := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; _, _ = w.Write([]byte(`{"status":"healthy"}`)) }))
	defer target.Close()
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		status  string
	}{
		{"healthy", func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != "" || r.Header.Get("X-Signature") != "" || r.Header.Get("Cookie") != "" {
				t.Error("business credential transmitted")
			}
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
		}, "pass"},
		{"redirect", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }, "fail"},
		{"oversize", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", maxHealthBytes+1)))
		}, "fail"},
		{"html proxy", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>sign in</html>")) }, "fail"},
		{"unknown JSON", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"secret":"never-print-this"}`)) }, "unknown"},
		{"guarded", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }, "fail"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()
			r := Report{}
			probeHealth(context.Background(), healthTarget{id: "probe", base: server.URL, path: "/healthz"}, &r)
			if r.Checks[0].Status != tc.status {
				t.Fatalf("status=%s", r.Checks[0].Status)
			}
			data, _ := json.Marshal(r)
			if bytes.Contains(data, []byte("never-print-this")) {
				t.Fatal("raw body exposed")
			}
		})
	}
	if hits != 0 {
		t.Fatal("redirect followed")
	}
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"status":"healthy"}`)) }))
	defer tlsServer.Close()
	r := Report{}
	probeHealth(context.Background(), healthTarget{id: "tls", base: tlsServer.URL, path: "/healthz"}, &r)
	if r.Checks[0].Status != "fail" {
		t.Fatal("untrusted TLS accepted")
	}
	caFile := t.TempDir() + "/ca.pem"
	if err := os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsServer.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	r = Report{}
	probeHealth(context.Background(), healthTarget{id: "trusted-tls", base: tlsServer.URL, path: "/healthz", prefix: "SERVICE", values: map[string]string{"SERVICE_TLS_CA_CERT_FILE": caFile}}, &r)
	if r.Checks[0].Status != "pass" {
		t.Fatal("configured private CA was not honored")
	}
}

func TestEnvironmentScopeAndExitStatuses(t *testing.T) {
	values := map[string]string{"AUTH_HOST": "auth.example.test", "PASSWORDS": "plaintext:synthetic-password", "GH_TOKEN": "never-read", "SESSION_EXCHANGE_SECRET": "synthetic-exchange-secret-123456789"}
	read := map[string]bool{}
	lookup := func(key string) (string, bool) { read[key] = true; v, ok := values[key]; return v, ok }
	var out, errOut bytes.Buffer
	if code := Run([]string{"--json"}, lookup, &out, &errOut); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if read["GH_TOKEN"] || strings.Contains(out.String(), "synthetic-password") {
		t.Fatal("environment scope or privacy violated")
	}
	var report Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Mode != "offline" {
		t.Fatal("offline mode changed")
	}
	m := fixture()
	m.Warden = nil
	if Inspect(m).ExitCode() != 3 {
		t.Fatal("missing server config reported as passed")
	}
	out.Reset()
	if Run([]string{"--help"}, lookup, &out, &errOut) != 0 || !strings.Contains(out.String(), "Usage:") {
		t.Fatal("help unavailable")
	}
}

func TestCredentialFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission contract")
	}
	m := fixture()
	data, _ := json.Marshal(m)
	path := t.TempDir() + "/private-name.json"
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil || strings.Contains(err.Error(), path) {
		t.Fatal("credential file permissions not enforced privately")
	}
	if err := os.WriteFile(path, []byte(`{"gateway":{"AUTH_HOST":"auth.example.test","PASSWORDS":""}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err != nil {
		t.Fatal("noncredential manifest rejected")
	}
	if _, err := LoadFile(t.TempDir()); err == nil {
		t.Fatal("directory accepted")
	}
}

func TestCommandHealthUsesOnlyConfiguredReadinessRoutes(t *testing.T) {
	paths := map[string]int{}
	var pathsMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathsMu.Lock()
		paths[r.URL.Path]++
		pathsMu.Unlock()
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Signature") != "" || r.Header.Get("Cookie") != "" {
			t.Error("business authentication was transmitted")
		}
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()
	m := fixture()
	m.GatewayURL = server.URL
	m.Gateway["AUTH_HOST"] = strings.TrimPrefix(server.URL, "http://")
	m.Gateway["COOKIE_SECURE"] = "false"
	m.Gateway["WARDEN_URL"] = server.URL
	m.Gateway["HERALD_ENABLED"] = "true"
	m.Gateway["HERALD_URL"] = server.URL
	m.Gateway["HERALD_HMAC_SECRET"] = "synthetic-herald-secret"
	m.Gateway["HERALD_HMAC_KEY_ID"] = "caller"
	m.Gateway["HERALD_TOTP_ENABLED"] = "true"
	m.Herald = map[string]string{"HERALD_HMAC_KEYS": `{"caller":"synthetic-herald-secret"}`, "HERALD_TOTP_BASE_URL": server.URL + "/provider", "HERALD_TOTP_HMAC_SECRET": "synthetic-totp-secret"}
	m.TOTP = map[string]string{"HERALD_TOTP_HMAC_KEYS": `{"sole":"synthetic-totp-secret"}`}
	data, _ := json.Marshal(m)
	file := t.TempDir() + "/config.json"
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := Run([]string{"--config", file, "--health", "--json"}, func(string) (string, bool) { panic("environment inspected") }, &out, &errOut); code != 0 {
		t.Fatalf("exit=%d output=%s", code, out.String())
	}
	pathsMu.Lock()
	defer pathsMu.Unlock()
	if paths["/readyz"] != 2 || paths["/health"] != 1 || paths["/provider/healthz"] != 1 || len(paths) != 3 {
		t.Fatal("unexpected probe route")
	}
	for _, value := range []string{"synthetic-herald-secret", "synthetic-totp-secret", file, server.URL} {
		if strings.Contains(out.String()+errOut.String(), value) {
			t.Fatal("health diagnostic exposed an input")
		}
	}
}

func TestHealthUnknownTargetsAndCancelledContext(t *testing.T) {
	m := fixture()
	m.Gateway["WARDEN_ENABLED"] = "false"
	r := Inspect(m)
	Health(context.Background(), m, &r)
	if find(r, "gateway.health").Status != "unknown" {
		t.Fatal("missing public URL inferred")
	}
	m.GatewayURL = "https://other.example.test"
	r = Report{}
	Health(context.Background(), m, &r)
	if find(r, "gateway.health").Status != "fail" {
		t.Fatal("different public authority accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r = Report{}
	probeHealth(ctx, healthTarget{id: "cancelled", base: "https://unreachable.invalid", path: "/healthz"}, &r)
	if r.Checks[0].Status != "fail" {
		t.Fatal("cancelled health check passed")
	}
	r = Report{}
	probeHealth(context.Background(), healthTarget{id: "bad-tls-file", base: "https://unreachable.invalid", prefix: "SERVICE", values: map[string]string{"SERVICE_TLS_CA_CERT_FILE": "private-do-not-echo"}}, &r)
	if r.Checks[0].Status != "fail" || strings.Contains(r.Checks[0].Message, "private-do-not-echo") {
		t.Fatal("TLS file error exposed a path")
	}
}
