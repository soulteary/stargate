package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	logger "github.com/soulteary/logger-kit/v3"
	"github.com/stretchr/testify/require"
)

// requestLogLine sends one login request through the full application and
// returns the request log line it produced.
func requestLogLine(t *testing.T, trustedProxies, forwardedFor string) map[string]any {
	t.Helper()
	ensureTestWorkingDir(t)
	t.Setenv("SESSION_STORAGE_ENABLED", "false")
	t.Setenv("TRUSTED_PROXIES", trustedProxies)
	setupTestConfig(t)

	var output bytes.Buffer
	previous := log
	t.Cleanup(func() { log = previous })
	log = logger.New(logger.Config{Level: logger.InfoLevel, Format: logger.FormatJSON, Output: &output})

	app := createApp()
	req := httptest.NewRequest(http.MethodPost, RouteLogin, strings.NewReader("password=wrong"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", forwardedFor)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		var line map[string]any
		if json.Unmarshal(scanner.Bytes(), &line) == nil && line["message"] == "HTTP request" {
			return line
		}
	}
	t.Fatalf("no request log line in %q", output.String())
	return nil
}

// Behind a trusted proxy the request log must name the same client as the
// audit log and the rate limiter, which all use the address Fiber resolves
// from TRUSTED_PROXIES. The proxy here appends to X-Forwarded-For, so the
// leftmost element is what the client sent and must not be the one logged.
func TestRequestLogCarriesTheResolvedClientAddress(t *testing.T) {
	// app.Test connects from 0.0.0.0.
	line := requestLogLine(t, "0.0.0.0", "198.51.100.9, 203.0.113.7")

	require.Equal(t, "203.0.113.7", line["client_ip"])
	require.Equal(t, "0.0.0.0", line["ip"])
}

// Without a trusted proxy the forwarded header is not believed, as everywhere
// else.
func TestRequestLogIgnoresForwardedAddressFromUntrustedPeer(t *testing.T) {
	line := requestLogLine(t, "", "203.0.113.7")

	require.Equal(t, "0.0.0.0", line["client_ip"])
	require.Equal(t, "0.0.0.0", line["ip"])
}
