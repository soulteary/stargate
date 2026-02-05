package handlers

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

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
	t.Setenv("HERALD_TOTP_ENABLED", "true")
	t.Setenv("HERALD_TOTP_BASE_URL", "http://localhost")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)
	InitHeraldTOTPClient(testLogger())

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
	ResetHeraldTOTPClientForTest()
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
	t.Setenv("HERALD_TOTP_ENABLED", "true")
	t.Setenv("HERALD_TOTP_BASE_URL", "http://localhost")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)
	InitHeraldTOTPClient(testLogger())

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
	ResetHeraldTOTPClientForTest()
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
