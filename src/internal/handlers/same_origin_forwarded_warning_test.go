package handlers

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

const sameOriginForwardedWarningText = "Rejected a same-origin request as cross-site because forwarded headers from an untrusted peer were ignored"

func sameOriginWarningApp(t *testing.T, trustedProxies string) (*fiber.App, *bytes.Buffer) {
	t.Helper()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("TRUSTED_PROXIES", trustedProxies)
	require.NoError(t, config.Initialize(testLogger()))

	output := &bytes.Buffer{}
	previousLog := log
	t.Cleanup(func() { log = previousLog })
	log = logger.New(logger.Config{Level: logger.WarnLevel, Format: logger.FormatJSON, Output: output})
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	proxies := []string{}
	if trustedProxies != "" {
		proxies = append(proxies, trustedProxies)
	}
	app := fiber.New(fiber.Config{TrustProxy: true, TrustProxyConfig: fiber.TrustProxyConfig{Proxies: proxies}})
	app.Post("/_login", RequireSameOrigin(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	return app, output
}

func sameOriginWarningStatus(t *testing.T, app *fiber.App, headers map[string]string) int {
	t.Helper()
	request := httptest.NewRequest(fiber.MethodPost, "http://auth.example.com/_login", nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := app.Test(request)
	require.NoError(t, err)
	return response.StatusCode
}

// Behind a proxy that terminates HTTPS and is missing from TRUSTED_PROXIES,
// the browser's own sign-in is rejected as cross-site. The response does not
// say why, so the request that shows the symptom has to log the cause.
func TestRequireSameOriginWarnsWhenIgnoredForwardedHeadersCauseTheRejection(t *testing.T) {
	app, output := sameOriginWarningApp(t, "")
	browser := map[string]string{
		"Origin":            "https://auth.example.com",
		"Sec-Fetch-Site":    "same-origin",
		"X-Forwarded-Proto": "https",
		"X-Forwarded-Host":  "auth.example.com",
		"X-Forwarded-For":   "203.0.113.7",
	}

	require.Equal(t, fiber.StatusForbidden, sameOriginWarningStatus(t, app, browser))
	logged := output.String()
	require.Contains(t, logged, sameOriginForwardedWarningText)
	for _, part := range []string{"TRUSTED_PROXIES", "answered 403", `"path":"/_login"`} {
		require.Contains(t, logged, part)
	}

	// Once per process, not once per request.
	require.Equal(t, fiber.StatusForbidden, sameOriginWarningStatus(t, app, browser))
	require.Equal(t, 1, strings.Count(output.String(), sameOriginForwardedWarningText))
}

func TestRequireSameOriginDoesNotWarnForOtherRejections(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		// A real cross-site request: the forwarded headers do not describe
		// the origin it claims.
		"cross-site origin": {
			"Origin":            "https://evil.example.net",
			"X-Forwarded-Proto": "https",
			"X-Forwarded-Host":  "auth.example.com",
		},
		// No proxy involved, so TRUSTED_PROXIES is not the cause.
		"no forwarded headers": {
			"Origin": "https://auth.example.com",
		},
	} {
		t.Run(name, func(t *testing.T) {
			app, output := sameOriginWarningApp(t, "")

			require.Equal(t, fiber.StatusForbidden, sameOriginWarningStatus(t, app, headers))
			require.NotContains(t, output.String(), sameOriginForwardedWarningText)
		})
	}
}

func TestRequireSameOriginAcceptsForwardedOriginFromTrustedProxy(t *testing.T) {
	app, output := sameOriginWarningApp(t, "0.0.0.0")

	require.Equal(t, fiber.StatusNoContent, sameOriginWarningStatus(t, app, map[string]string{
		"Origin":            "https://auth.example.com",
		"X-Forwarded-Proto": "https",
		"X-Forwarded-Host":  "auth.example.com",
	}))
	require.NotContains(t, output.String(), sameOriginForwardedWarningText)
}
