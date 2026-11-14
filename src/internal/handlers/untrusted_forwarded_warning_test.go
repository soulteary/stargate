package handlers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

// Behind a proxy that is missing from TRUSTED_PROXIES, /_auth redirects to a
// login URL without the callback and with the wrong scheme. The request that
// shows the symptom must also log the cause, and the text must name it.
func TestForwardAuthRedirectWarnsAboutIgnoredForwardedHeaders(t *testing.T) {
	isolateGlobalState(t)
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("CALLBACK_ALLOWED_HOSTS", "app.example.com")
	t.Setenv("SESSION_EXCHANGE_SECRET", "0123456789abcdef0123456789abcdef")
	// The package's tests trust the synthetic peer by default.
	t.Setenv("TRUSTED_PROXIES", "")
	require.NoError(t, config.Initialize(testLogger()))

	var output bytes.Buffer
	previousLog := log
	t.Cleanup(func() { log = previousLog })
	InitForwardAuthHandler(logger.New(logger.Config{Level: logger.WarnLevel, Format: logger.FormatJSON, Output: &output}))
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	handler := CheckRoute(setupTestStore())
	request := func() string {
		ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
			"Accept":            "text/html",
			"Host":              "stargate:8080",
			"X-Forwarded-Host":  "app.example.com",
			"X-Forwarded-Proto": "https",
			"X-Forwarded-For":   "203.0.113.7",
		}, "")
		defer app.ReleaseCtx(ctx)
		require.NoError(t, handler(ctx))
		require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
		return ctx.GetRespHeader(fiber.HeaderLocation)
	}

	// The symptom: no callback, and http although the user came over https.
	require.Equal(t, "http://auth.example.com/_login", request())

	logged := output.String()
	require.Contains(t, logged, "Received forwarded headers from an untrusted peer")
	for _, effect := range []string{"loses its callback", "uses http://", "not returned to the application", "every client shares one quota", "TRUSTED_PROXIES"} {
		require.Contains(t, logged, effect)
	}

	// Once per process, not once per request.
	request()
	require.Equal(t, 1, strings.Count(output.String(), "Received forwarded headers from an untrusted peer"))
}

func TestForwardAuthRedirectDoesNotWarnForTrustedProxy(t *testing.T) {
	isolateGlobalState(t)
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("CALLBACK_ALLOWED_HOSTS", "app.example.com")
	t.Setenv("SESSION_EXCHANGE_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("TRUSTED_PROXIES", "0.0.0.0")
	require.NoError(t, config.Initialize(testLogger()))

	var output bytes.Buffer
	previousLog := log
	t.Cleanup(func() { log = previousLog })
	InitForwardAuthHandler(logger.New(logger.Config{Level: logger.WarnLevel, Format: logger.FormatJSON, Output: &output}))
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{
		"Accept":            "text/html",
		"X-Forwarded-Host":  "app.example.com",
		"X-Forwarded-Proto": "https",
		"X-Forwarded-For":   "203.0.113.7",
	}, "")
	defer app.ReleaseCtx(ctx)
	require.NoError(t, CheckRoute(setupTestStore())(ctx))
	require.Equal(t, "https://auth.example.com/_login?callback=app.example.com", ctx.GetRespHeader(fiber.HeaderLocation))
	require.NotContains(t, output.String(), "untrusted peer")
}
