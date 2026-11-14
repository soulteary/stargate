package handlers

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func setupStepUpReturnTest(t *testing.T, cookieSecure string) *fakeHerald {
	t.Helper()
	fake := setupReauthDeployment(t)
	setConfigValue(t, &config.AuthHost, "auth.example.com")
	setConfigValue(t, &config.CallbackAllowedHosts, "app.example.com")
	setConfigValue(t, &config.CookieDomain, "")
	setConfigValue(t, &config.CookieSecure, cookieSecure)
	fake.totpEnabled["alice"], fake.totpCodes["alice"] = true, "111111"
	return fake
}

// With COOKIE_SECURE=true a step-up link that asks to return over http must not
// end in a redirect to a plaintext URL, whatever the case or padding of the
// submitted scheme. The return falls back to the local path, as it does for
// any other scheme that is not accepted.
func TestStepUpReturnRefusesHTTPWhenCookieSecure(t *testing.T) {
	setupStepUpReturnTest(t, "true")
	store := setupTestStore()

	for _, proto := range []string{"http", "HTTP", "%20http%20"} {
		alice := wardenReauthSession(t, store, "alice")
		ctx := reauthRequest(t, alice, "POST", "/_step_up", "code=111111&callback=/admin/users&callback_host=app.example.com&callback_proto="+proto, StepUpAPI(store))
		require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode(), proto)
		require.Equal(t, "/admin/users", string(ctx.Response().Header.Peek("Location")), proto)
		require.NotNil(t, reauthSessionValue(t, store, alice, "step_up_verified_at"), proto)

		// The form does not carry the refused scheme into the POST either.
		page := reauthRequest(t, alice, "GET", "/_step_up?callback=/admin/users&callback_host=app.example.com&callback_proto="+proto, "", StepUpRoute(store))
		require.Equal(t, fiber.StatusOK, page.Response().StatusCode(), proto)
		require.Contains(t, string(page.Response().Body()), `name="callback_proto" value=""`, proto)
	}
	require.Equal(t, "/admin", stepUpReturnURL("http", "app.example.com", "/admin"))
}

func TestStepUpReturnKeepsHTTPSWhenCookieSecure(t *testing.T) {
	setupStepUpReturnTest(t, "true")
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")

	page := reauthRequest(t, alice, "GET", "/_step_up?callback=/admin/users&callback_host=app.example.com&callback_proto=https", "", StepUpRoute(store))
	require.Contains(t, string(page.Response().Body()), `name="callback_proto" value="https"`)
	ctx := reauthRequest(t, alice, "POST", "/_step_up", "code=111111&callback=/admin/users&callback_host=app.example.com&callback_proto=https", StepUpAPI(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.Equal(t, "https://app.example.com/admin/users", string(ctx.Response().Header.Peek("Location")))
}

// A local deployment without TLS sets COOKIE_SECURE=false and still returns to
// the application over http.
func TestStepUpReturnKeepsHTTPWhenCookieNotSecure(t *testing.T) {
	setupStepUpReturnTest(t, "false")
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")

	page := reauthRequest(t, alice, "GET", "/_step_up?callback=/admin/users&callback_host=app.example.com&callback_proto=http", "", StepUpRoute(store))
	require.Contains(t, string(page.Response().Body()), `name="callback_proto" value="http"`)
	ctx := reauthRequest(t, alice, "POST", "/_step_up", "code=111111&callback=/admin/users&callback_host=app.example.com&callback_proto=http", StepUpAPI(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.Equal(t, "http://app.example.com/admin/users", string(ctx.Response().Header.Peek("Location")))
	require.Equal(t, "https://app.example.com/admin", stepUpReturnURL("https", "app.example.com", "/admin"))
	require.Equal(t, "/admin", stepUpReturnURL("ftp", "app.example.com", "/admin"))
}
