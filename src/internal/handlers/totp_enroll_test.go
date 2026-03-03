package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/heraldtotp"
	"github.com/soulteary/stargate/src/internal/i18n"
)

func TestTOTPEnrollRoute_NotAuthenticated_Redirects(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPEnrollRoute(store)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI("/totp/enroll")
	ctx.Request().Header.SetMethod("GET")
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusFound, ctx.Response().StatusCode())
	testza.AssertEqual(t, "/_login", string(ctx.Response().Header.Peek("Location")))
}

func TestTOTPEnrollRoute_Authenticated_NoUserID_400(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPEnrollRoute(store)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI("/totp/enroll")
	ctx.Request().Header.SetMethod("GET")
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)
	// Do not set user_id so handler returns 400

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
}

func TestTOTPEnrollRoute_Authenticated_ClientNil_503(t *testing.T) {
	ResetHeraldTOTPClientForTest()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	// Do not set HERALD_TOTP_ENABLED so getHeraldTOTPClient() stays nil
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPEnrollRoute(store)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI("/totp/enroll")
	ctx.Request().Header.SetMethod("GET")
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)
	ctx.Request().Header.Set("Cookie", auth.SessionCookieName+"="+sess.ID())

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusServiceUnavailable, ctx.Response().StatusCode())
}

func TestTOTPEnrollRoute_AlreadyBound_RedirectsToRevoke(t *testing.T) {
	// Mock herald-totp: GET /v1/status returns totp_enabled true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(heraldtotp.StatusResponse{
			Subject:     "u_test",
			TotpEnabled: true,
		})
	}))
	defer server.Close()

	ResetHeraldTOTPClientForTest()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("HERALD_TOTP_ENABLED", "true")
	t.Setenv("HERALD_TOTP_BASE_URL", server.URL)
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)
	InitHeraldTOTPClient(testLogger())

	store := setupTestStore()
	handler := TOTPEnrollRoute(store)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI("/totp/enroll")
	ctx.Request().Header.SetMethod("GET")
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)
	ctx.Request().Header.Set("Cookie", auth.SessionCookieName+"="+sess.ID())

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusFound, ctx.Response().StatusCode())
	testza.AssertEqual(t, "/totp/revoke", string(ctx.Response().Header.Peek("Location")))
}

func TestTOTPEnrollConfirmAPI_NotAuthenticated_401(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPEnrollConfirmAPI(store)

	ctx, app := createTestContext("POST", "/totp/enroll/confirm", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "enroll_id=e1&code=123456")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func TestTOTPEnrollConfirmAPI_MissingParams_400(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPEnrollConfirmAPI(store)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI("/totp/enroll/confirm")
	ctx.Request().Header.SetMethod("POST")
	ctx.Request().Header.SetContentType("application/x-www-form-urlencoded")
	ctx.Request().SetBodyString("enroll_id=e1")
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")
	ctx.Request().Header.Set("Cookie", auth.SessionCookieName+"="+sess.ID())

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
}

func TestTOTPEnrollConfirmAPI_ClientNil_503(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := TOTPEnrollConfirmAPI(store)

	ctx, app := createTestContext("POST", "/totp/enroll/confirm", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "enroll_id=e1&code=123456")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")
	ctx.Request().Header.Set("Cookie", auth.SessionCookieName+"="+sess.ID())

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusServiceUnavailable, ctx.Response().StatusCode())
}
