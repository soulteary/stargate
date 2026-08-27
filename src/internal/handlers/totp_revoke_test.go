package handlers

import (
	"errors"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"

	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

func TestRevokeErrorReason(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", want: ""},
		{name: "http unauthorized", err: errors.New("401"), want: "unauthorized"},
		{name: "text unauthorized", err: errors.New("unauthorized"), want: "unauthorized"},
		{name: "missing route", err: errors.New("Cannot POST /totp/revoke"), want: "unavailable"},
		{name: "rate limited", err: errors.New("rate_limit exceeded"), want: "rate_limited"},
		{name: "bad gateway", err: errors.New("502 upstream failure"), want: "service_unavailable"},
		{name: "connection", err: errors.New("connection refused"), want: "service_unavailable"},
		{name: "unknown", err: errors.New("invalid response"), want: "service_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testza.AssertEqual(t, tt.want, revokeErrorReason(tt.err))
		})
	}
}

func TestTOTPRevokeRoute_NotAuthenticated_Redirects(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPRevokeRoute(store)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI("/totp/revoke")
	ctx.Request().Header.SetMethod("GET")
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusFound, ctx.Response().StatusCode())
	testza.AssertEqual(t, "/_login", string(ctx.Response().Header.Peek("Location")))
}

func TestTOTPRevokeRoute_Authenticated_NoUserID_400(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_URL", "http://localhost")
	t.Setenv("WARDEN_ENABLED", "true")
	t.Setenv("WARDEN_URL", "http://warden.test")
	t.Setenv("HERALD_TOTP_ENABLED", "true")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, InitHeraldClient(testLogger()))

	store := setupTestStore()
	handler := TOTPRevokeRoute(store)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI("/totp/revoke")
	ctx.Request().Header.SetMethod("GET")
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
}

func TestTOTPRevokeRoute_Authenticated_ClientNil_503(t *testing.T) {
	ResetHeraldClientForTest()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPRevokeRoute(store)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI("/totp/revoke")
	ctx.Request().Header.SetMethod("GET")
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusServiceUnavailable, ctx.Response().StatusCode())
}

func TestTOTPRevokeConfirmAPI_NotAuthenticated_401(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPRevokeConfirmAPI(store)

	ctx, app := createTestContext("POST", "/totp/revoke", nil, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func TestTOTPRevokeConfirmAPI_NoUserID_400(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_URL", "http://localhost")
	t.Setenv("WARDEN_ENABLED", "true")
	t.Setenv("WARDEN_URL", "http://warden.test")
	t.Setenv("HERALD_TOTP_ENABLED", "true")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, InitHeraldClient(testLogger()))

	store := setupTestStore()
	handler := TOTPRevokeConfirmAPI(store)

	ctx, app := createTestContext("POST", "/totp/revoke", nil, "")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)
	// Do not set user_id

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
}

func TestTOTPRevokeConfirmAPI_ClientNil_503(t *testing.T) {
	ResetHeraldClientForTest()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPRevokeConfirmAPI(store)

	ctx, app := createTestContext("POST", "/totp/revoke", nil, "")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusServiceUnavailable, ctx.Response().StatusCode())
}

func TestTOTPRevokeConfirmAPI_RequiresRecentAuthentication(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	testza.AssertNoError(t, config.Initialize(testLogger()))
	originalClient := heraldClient
	heraldClient = &herald.Client{}
	t.Cleanup(func() { heraldClient = originalClient })

	store := setupTestStore()
	ctx, app := createTestContext("POST", "/totp/revoke", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")
	testza.AssertNoError(t, auth.Authenticate(sess))

	err = TOTPRevokeConfirmAPI(store)(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	testza.AssertContains(t, string(ctx.Response().Body()), "recent_authentication_required")
}
