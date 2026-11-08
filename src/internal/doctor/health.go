package doctor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/tlsconfig"
)

const maxHealthBytes = 64 << 10

type healthTarget struct {
	id, base, path, prefix string
	values                 map[string]string
}

// Health performs only bounded health GETs. It never sends API keys, HMAC or browser cookies.
func Health(ctx context.Context, m Manifest, r *Report) {
	r.Mode = "health"
	r.Limits = append(r.Limits, "Health GETs may produce server access logs/metrics; no authenticated business request or Redis write-permission probe is sent.")
	targets := []healthTarget{}
	if m.GatewayURL == "" {
		r.add("gateway.health", "unknown", "Supply gateway_url or --gateway-url for an explicit public health target; the scheme is not inferred from COOKIE_SECURE.")
	} else {
		u, err := url.Parse(m.GatewayURL)
		if err != nil || !validRoot(m.GatewayURL) || !strings.EqualFold(u.Host, m.Gateway["AUTH_HOST"]) {
			r.add("gateway.health", "fail", "The gateway health root must match AUTH_HOST and contain no credentials, path, query or fragment.")
		} else {
			targets = append(targets, healthTarget{"gateway.health", m.GatewayURL, "/readyz", "GATEWAY", map[string]string{"GATEWAY_TLS_CA_CERT_FILE": m.GatewayCAFile}})
		}
	}
	for _, service := range []string{"WARDEN", "HERALD"} {
		if enabled, ok := boolean(m.Gateway, service+"_ENABLED", false); enabled && ok && validRoot(m.Gateway[service+"_URL"]) {
			path := "/health"
			if service == "HERALD" {
				path = "/readyz"
			}
			targets = append(targets, healthTarget{strings.ToLower(service) + ".health", m.Gateway[service+"_URL"], path, service, m.Gateway})
		}
	}
	if enabled, ok := boolean(m.Gateway, "HERALD_TOTP_ENABLED", false); enabled && ok && validProxyURL(m.Herald["HERALD_TOTP_BASE_URL"]) {
		targets = append(targets, healthTarget{"totp.health", m.Herald["HERALD_TOTP_BASE_URL"], "/healthz", "HERALD_TOTP", m.Herald})
	}
	for _, target := range targets {
		probeHealth(ctx, target, r)
	}
}

func validRoot(raw string) bool {
	return strings.TrimSpace(raw) != "" && strings.TrimSpace(raw) == raw && !strings.ContainsAny(raw, "?#") && config.ValidateServiceRoot(raw) == nil
}

func validProxyURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Hostname() != "" && (u.Scheme == "https" || u.Scheme == "http") && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && !strings.ContainsAny(raw, "?#\r\n\t\\")
}

func probeHealth(ctx context.Context, t healthTarget, r *Report) {
	if !validProxyURL(t.base) {
		r.add(t.id, "fail", "The configured health target contains invalid URL components.")
		return
	}
	settings := t.values
	for _, suffix := range []string{"_TLS_CA_CERT_FILE", "_TLS_CLIENT_CERT_FILE", "_TLS_CLIENT_KEY_FILE"} {
		if path := settings[t.prefix+suffix]; path != "" {
			info, err := os.Stat(path)
			if err != nil || !info.Mode().IsRegular() || info.Size() > maxConfigBytes {
				r.add(t.id, "fail", "Configured TLS files must be readable regular files within the size limit.")
				return
			}
		}
	}
	tlsConfig, err := tlsconfig.Build("diagnostic", settings[t.prefix+"_TLS_CA_CERT_FILE"], settings[t.prefix+"_TLS_CLIENT_CERT_FILE"], settings[t.prefix+"_TLS_CLIENT_KEY_FILE"], settings[t.prefix+"_TLS_SERVER_NAME"])
	if err != nil {
		r.add(t.id, "fail", "Cannot load the configured TLS material; check CA, client certificate/key and server-name settings.")
		return
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = tlsConfig
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: 2 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	target, _ := url.Parse(t.base)
	target.Path = strings.TrimRight(target.Path, "/") + t.path
	target.RawPath = ""
	target.RawQuery = ""
	target.ForceQuery = false
	target.Fragment = ""
	target.RawFragment = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		r.add(t.id, "fail", "The configured health target is invalid.")
		return
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		r.add(t.id, "fail", "Health transport failed; check DNS, network, TLS trust/server name and client certificate requirements.")
		return
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxHealthBytes+1))
	if err != nil || len(data) > maxHealthBytes {
		r.add(t.id, "fail", "Health response is unreadable or exceeds the size limit.")
		return
	}
	if response.StatusCode != http.StatusOK {
		r.add(t.id, "fail", "Health endpoint did not return HTTP 200; check readiness, Redis and health access policy.")
		return
	}
	var body map[string]json.RawMessage
	if json.Unmarshal(data, &body) != nil || body == nil {
		r.add(t.id, "fail", "The health endpoint did not return a JSON object; check the proxy route.")
		return
	}
	var status string
	if json.Unmarshal(body["status"], &status) != nil || (status != "healthy" && status != "ready" && status != "ok") {
		r.add(t.id, "unknown", "The response is not a recognized healthy/readiness result; inspect the endpoint locally without publishing its raw body.")
		return
	}
	r.add(t.id, "pass", "Configured health endpoint reports ready/healthy; service authentication and business operations remain untested.")
}
