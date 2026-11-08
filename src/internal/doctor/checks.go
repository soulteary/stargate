package doctor

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/url"
	"strings"

	"github.com/soulteary/stargate/src/internal/config"
)

// Check describes a bounded diagnostic without configuration values or raw errors.
type Check struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Report describes the requested checks, not proof of a working login flow.
type Report struct {
	Mode   string   `json:"mode"`
	Checks []Check  `json:"checks"`
	Limits []string `json:"limits"`
}

func (r *Report) add(id, status, message string) {
	r.Checks = append(r.Checks, Check{id, status, message})
}
func result(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}
func boolean(values map[string]string, key string, defaultValue bool) (bool, bool) {
	value, ok := values[key]
	if !ok {
		return defaultValue, true
	}
	if strings.EqualFold(value, "true") {
		return true, true
	}
	if strings.EqualFold(value, "false") {
		return false, true
	}
	return false, false
}

// Inspect checks a documented subset of configuration using pure startup validators.
func Inspect(m Manifest) Report {
	r := Report{Mode: "offline", Checks: []Check{}, Limits: []string{
		"Scoped configuration inspection, not full application startup validation or a login test; UI settings and service action policies are outside scope.",
		"Cookie transmission, callback routing, runtime versions and service authentication are not proven offline.",
		"Health checks do not prove HMAC, Redis write permissions, provider delivery or TOTP recovery.",
		"Authenticated GET requests can write replay nonces and audit records; active probes are not implemented.",
	}}
	g := m.Gateway
	wardenEnabled, _ := boolean(g, "WARDEN_ENABLED", false)
	passwordsOK := config.ValidatePasswordsOrEmpty(config.EnvVariable{Value: g["PASSWORDS"]}) && (wardenEnabled || g["PASSWORDS"] != "")
	r.add("gateway.password_config", result(passwordsOK), "Check PASSWORDS syntax with the startup parser; a password-only deployment requires a configured password. No password is authenticated.")
	r.add("gateway.host_cookie", result(config.ValidateHostCookieDomain(g["AUTH_HOST"], g["COOKIE_DOMAIN"]) == nil), "Check AUTH_HOST and COOKIE_DOMAIN using the startup host/domain contract.")
	secure, valid := boolean(g, "COOKIE_SECURE", true)
	r.add("gateway.cookie_secure", result(valid), "COOKIE_SECURE must be true or false; actual browser transport is not tested.")
	if m.GatewayURL != "" {
		u, err := url.Parse(m.GatewayURL)
		if err != nil || !validRoot(m.GatewayURL) || !strings.EqualFold(u.Host, g["AUTH_HOST"]) {
			r.add("gateway.public_url", "fail", "The explicit public URL must be a service root matching AUTH_HOST.")
		} else {
			r.add("gateway.public_url", "pass", "The explicit public URL matches AUTH_HOST; actual proxy routing remains untested.")
			if secure && u.Scheme == "http" {
				status := "fail"
				ip := net.ParseIP(u.Hostname())
				if u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()) {
					status = "unknown"
				}
				r.add("gateway.cookie_transport", status, "Secure cookies require a suitable secure browser origin; check the public HTTPS scheme and TLS termination. Localhost exceptions require browser verification.")
			}
		}
	}
	r.add("gateway.callbacks", result(config.ValidateCallbackHosts(g["CALLBACK_ALLOWED_HOSTS"]) == nil), "Check CALLBACK_ALLOWED_HOSTS syntax; reachability and redirect authorization are not tested.")
	path := g["SESSION_EXCHANGE_REDIRECT_PATH"]
	if path == "" {
		path = "/"
	}
	r.add("gateway.exchange_path", result(config.ValidateSessionExchangeRedirectPath(config.EnvVariable{Value: path})), "Check SESSION_EXCHANGE_REDIRECT_PATH using the startup validator.")
	if g["SESSION_EXCHANGE_SECRET"] == "" {
		r.add("gateway.exchange_secret", "skipped", "Session exchange is not configured; this is not an application startup failure.")
	} else if len(g["SESSION_EXCHANGE_SECRET"]) < 32 {
		r.add("gateway.exchange_secret", "fail", "SESSION_EXCHANGE_SECRET must contain at least 32 characters when configured.")
	} else {
		r.add("gateway.exchange_secret", "pass", "A session-exchange secret is supplied; replicas must use the same configured value.")
	}
	for _, service := range []string{"WARDEN", "HERALD"} {
		enabled, ok := boolean(g, service+"_ENABLED", false)
		r.add(strings.ToLower(service)+".enabled", result(ok), "Enable flags must be true or false.")
		if !enabled || !ok {
			continue
		}
		r.add(strings.ToLower(service)+".url", result(validRoot(g[service+"_URL"])), "Configure a valid service root URL; credentials, query and fragment are not allowed.")
		server := m.Warden
		if service == "HERALD" {
			server = m.Herald
		}
		inspectPair(&r, strings.ToLower(service)+".credentials", g, server, service, service == "HERALD")
	}
	if enabled, ok := boolean(g, "SESSION_STORAGE_ENABLED", false); !ok {
		r.add("gateway.redis", "fail", "SESSION_STORAGE_ENABLED must be true or false.")
	} else if enabled {
		address := g["SESSION_STORAGE_REDIS_ADDR"]
		if address == "" {
			address = config.SessionStorageRedisAddr.DefaultValue
		}
		_, _, err := net.SplitHostPort(address)
		r.add("gateway.redis_config", result(err == nil), "Check SESSION_STORAGE_REDIS_ADDR syntax; Redis connectivity and ACL write permissions are not tested offline.")
	}
	if enabled, ok := boolean(g, "HERALD_TOTP_ENABLED", false); !ok {
		r.add("totp.enabled", "fail", "HERALD_TOTP_ENABLED must be true or false.")
	} else if enabled {
		if m.Herald == nil {
			r.add("totp.credentials", "unknown", "Supply Herald proxy configuration to inspect the Herald-to-TOTP credential pair.")
		} else {
			r.add("totp.url", result(validProxyURL(m.Herald["HERALD_TOTP_BASE_URL"])), "Check HERALD_TOTP_BASE_URL; its configured path prefix is retained for health requests.")
			inspectPair(&r, "totp.credentials", m.Herald, m.TOTP, "HERALD_TOTP", false)
		}
	}
	return r
}

func inspectPair(r *Report, id string, client, server map[string]string, prefix string, herald bool) {
	if server == nil {
		r.add(id, "unknown", "Server configuration was not supplied; credential pairing cannot be checked.")
		return
	}
	if client[prefix+"_TLS_CLIENT_CERT_FILE"] != "" {
		r.add(id, "unknown", "mTLS is configured; certificate acceptance and authorization require separate runtime verification.")
		return
	}
	secret := client[prefix+"_HMAC_SECRET"]
	api := client[prefix+"_API_KEY"]
	if herald && server["REQUEST_AUTH_MODE"] == "api_key" && secret != "" {
		r.add(id, "fail", "The client selects HMAC while Herald is configured for API-key authentication; select a matching mode.")
		return
	}
	if secret != "" {
		kid := client[prefix+"_HMAC_KEY_ID"]
		keysName := "WARDEN_HMAC_KEYS"
		singleName := ""
		if herald {
			keysName = "HERALD_HMAC_KEYS"
			singleName = "HERALD_HMAC_SECRET"
		} else if prefix == "HERALD_TOTP" {
			keysName = "HERALD_TOTP_HMAC_KEYS"
			singleName = "HERALD_TOTP_HMAC_SECRET"
		}
		expected := ""
		var keys map[string]string
		if server[keysName] != "" {
			if json.Unmarshal([]byte(server[keysName]), &keys) != nil {
				r.add(id, "fail", "The server "+keysName+" key map is invalid JSON; no input values are shown.")
				return
			}
			if kid == "" && prefix == "HERALD_TOTP" && len(keys) == 1 {
				for _, value := range keys {
					expected = value
				}
			} else {
				if kid == "" {
					r.add(id, "fail", "Configure "+prefix+"_HMAC_KEY_ID for the mapped-key server.")
					return
				}
				expected = keys[kid]
			}
		} else if singleName != "" {
			expected = server[singleName]
			if herald && kid != "" && kid != "default" {
				expected = ""
			}
		}
		if herald && server["REQUEST_AUTH_MODE"] != "" && server["REQUEST_AUTH_MODE"] != "hmac_v2" {
			r.add(id, "unknown", "This Herald authentication mode is outside the diagnostic's supported scope.")
			return
		}
		if expected == "" {
			r.add(id, "fail", "Check "+prefix+"_HMAC_KEY_ID and the selected server "+keysName+"/single-secret configuration; no credential was resolved.")
			return
		}
		r.add(id, result(subtle.ConstantTimeCompare([]byte(secret), []byte(expected)) == 1), "Compare "+prefix+"_HMAC_SECRET with the selected server credential locally; no online authentication or service-policy authorization is proven.")
		return
	}
	if herald && (server["REQUEST_AUTH_MODE"] == "" || server["REQUEST_AUTH_MODE"] == "hmac_v2") {
		r.add(id, "fail", "Herald requires HMAC but the client has no selected HMAC credential.")
		return
	}
	r.add(id, result(api != "" && server["API_KEY"] != "" && subtle.ConstantTimeCompare([]byte(api), []byte(server["API_KEY"])) == 1), "Compare "+prefix+"_API_KEY with server API_KEY locally; runtime authorization is not tested.")
}

// ExitCode is zero for passed requested checks, one for failure, or three for incomplete checks.
func (r Report) ExitCode() int {
	unknown := false
	for _, c := range r.Checks {
		if c.Status == "fail" {
			return 1
		}
		unknown = unknown || c.Status == "unknown"
	}
	if unknown {
		return 3
	}
	return 0
}
