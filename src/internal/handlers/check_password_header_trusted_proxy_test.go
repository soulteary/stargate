package handlers

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// The synthetic test connection has the peer address 0.0.0.0.
const (
	testPeerAddress      = "0.0.0.0"
	testOtherProxy       = "10.0.0.1"
	requireTrustedProxy  = "PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY"
	unattributedResponse = "accepted only through a trusted proxy"
)

// setupPasswordHeaderProxyTest configures header authentication with the given
// TRUSTED_PROXIES and opt-in value and returns a function that sends one
// /_auth request from the synthetic peer and returns its status and body,
// together with the buffer warnings are written to.
func setupPasswordHeaderProxyTest(t *testing.T, trustedProxies, requireTrusted string) (func(password, forwardedFor string) (int, string), *bytes.Buffer, int) {
	t.Helper()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("PASSWORD_HEADER_AUTH_ENABLED", "true")
	t.Setenv("TRUSTED_PROXIES", trustedProxies)
	if requireTrusted != "" {
		t.Setenv(requireTrustedProxy, requireTrusted)
	}
	require.NoError(t, config.Initialize(testLogger()))

	output := &bytes.Buffer{}
	previousLog := log
	t.Cleanup(func() { log = previousLog })
	InitForwardAuthHandler(logger.New(logger.Config{Level: logger.WarnLevel, Format: logger.FormatJSON, Output: output}))
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	handler := CheckRoute(setupTestStore())
	// The application trusts the same proxies as the configuration, as the
	// server does.
	app := fiber.New(fiber.Config{
		TrustProxy:         true,
		TrustProxyConfig:   fiber.TrustProxyConfig{Proxies: strings.Split(trustedProxies, ",")},
		ProxyHeader:        config.ProxyHeader.String(),
		EnableIPValidation: true,
	})
	request := func(password, forwardedFor string) (int, string) {
		ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
		defer app.ReleaseCtx(ctx)
		ctx.Request().SetRequestURI("/_auth")
		ctx.Request().Header.SetMethod(fiber.MethodGet)
		ctx.Request().Header.Set("Accept", "application/json")
		ctx.Request().Header.Set("Stargate-Password", password)
		if forwardedFor != "" {
			ctx.Request().Header.Set("X-Forwarded-For", forwardedFor)
		}
		ctx.Locals("i18n-bundle", i18n.GetBundle())
		ctx.Locals("i18n-language", i18n.LangEN)
		require.NoError(t, handler(ctx))
		return ctx.Response().StatusCode(), string(ctx.Response().Body())
	}
	max := config.RateLimitLoginMax.ToInt(config.DefaultRateLimitLoginMax)
	require.Positive(t, max)
	return request, output, max
}

// passwordFailureCount returns how many attempts the store holds for address.
// It adds one attempt to read the counter, so call it last.
func passwordFailureCount(t *testing.T, address string) int {
	t.Helper()
	count, _, err := getRateLimitStore().Incr(context.Background(), "password-failure:"+address, rateLimitWindow())
	require.NoError(t, err)
	return count - 1
}

// With the opt-in, a peer outside TRUSTED_PROXIES gets no answer about the
// password and cannot use up a quota: wrong and correct values are refused
// alike, whatever client address the request claims, and nothing is counted.
func TestCheckRoute_RequireTrustedProxyRefusesUntrustedPeer(t *testing.T) {
	request, output, max := setupPasswordHeaderProxyTest(t, testOtherProxy, "true")

	for attempt := 1; attempt <= max+5; attempt++ {
		status, body := request("wrong-guess", "")
		require.Equal(t, fiber.StatusForbidden, status, "attempt %d", attempt)
		require.Contains(t, body, unattributedResponse)
	}
	status, _ := request("test123", "")
	require.Equal(t, fiber.StatusForbidden, status)
	status, _ = request("test123", "203.0.113.7")
	require.Equal(t, fiber.StatusForbidden, status)

	require.Zero(t, passwordFailureCount(t, testPeerAddress))
	require.Zero(t, passwordFailureCount(t, "203.0.113.7"))
	require.Equal(t, 1, strings.Count(output.String(), "the peer is not in TRUSTED_PROXIES"))
}

// A trusted proxy that sends no client address leaves every client at the
// proxy address, which is the same shared quota.
func TestCheckRoute_RequireTrustedProxyRefusesMissingClientAddress(t *testing.T) {
	request, output, _ := setupPasswordHeaderProxyTest(t, testPeerAddress, "true")

	for _, forwardedFor := range []string{"", testPeerAddress, "::ffff:" + testPeerAddress, "unknown"} {
		status, body := request("test123", forwardedFor)
		require.Equal(t, fiber.StatusForbidden, status, "X-Forwarded-For %q", forwardedFor)
		require.Contains(t, body, unattributedResponse)
	}
	require.Zero(t, passwordFailureCount(t, testPeerAddress))
	require.Equal(t, 1, strings.Count(output.String(), "supplied no client address other than its own"))
}

// Through a trusted proxy the quota is per client: one client that uses it up
// does not affect another, and the limit on guesses stays in force.
func TestCheckRoute_RequireTrustedProxyKeepsPerClientQuota(t *testing.T) {
	request, output, max := setupPasswordHeaderProxyTest(t, testPeerAddress, "true")

	for attempt := 1; attempt <= max; attempt++ {
		status, _ := request("wrong-guess", "203.0.113.7")
		require.Equal(t, fiber.StatusUnauthorized, status, "attempt %d", attempt)
	}
	status, _ := request("wrong-guess", "203.0.113.7")
	require.Equal(t, fiber.StatusTooManyRequests, status)
	status, _ = request("test123", "203.0.113.7")
	require.Equal(t, fiber.StatusTooManyRequests, status)

	status, _ = request("test123", "198.51.100.9")
	require.Equal(t, fiber.StatusOK, status)
	require.NotContains(t, output.String(), requireTrustedProxy)
}

// Without the opt-in nothing changes: an untrusted peer is still compared and
// counted at its own address.
func TestCheckRoute_UntrustedPeerIsComparedByDefault(t *testing.T) {
	for name, value := range map[string]string{"unset": "", "false": "false"} {
		t.Run(name, func(t *testing.T) {
			request, _, max := setupPasswordHeaderProxyTest(t, testOtherProxy, value)

			status, _ := request("test123", "")
			require.Equal(t, fiber.StatusOK, status)
			for attempt := 1; attempt <= max; attempt++ {
				status, _ := request("wrong-guess", "203.0.113.7")
				require.Equal(t, fiber.StatusUnauthorized, status, "attempt %d", attempt)
			}
			status, _ = request("test123", "")
			require.Equal(t, fiber.StatusTooManyRequests, status)
		})
	}
}
