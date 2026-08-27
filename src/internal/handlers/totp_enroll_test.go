package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"

	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

type failSetStorage struct {
	fiber.Storage
	fail bool
}

func (s *failSetStorage) Set(key string, value []byte, expiration time.Duration) error {
	if s.fail {
		return errors.New("injected session write failure")
	}
	return s.Storage.Set(key, value, expiration)
}

func (s *failSetStorage) SetWithContext(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if s.fail {
		return errors.New("injected session write failure")
	}
	return s.Storage.SetWithContext(ctx, key, value, expiration)
}

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
	ResetHeraldClientForTest()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	// Do not set HERALD_ENABLED / HERALD_URL so getHeraldClient() stays nil
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
	// Mock Herald TOTP proxy: GET /v1/totp/status returns totp_enabled true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/totp/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(herald.TOTPStatusResponse{
			Subject:     "u_test",
			TotpEnabled: true,
		})
	}))
	defer server.Close()

	ResetHeraldClientForTest()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_URL", server.URL)
	t.Setenv("HERALD_TOTP_ENABLED", "true")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, InitHeraldClient(testLogger()))

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
	ResetHeraldClientForTest()
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
	sess.Set("user_id", "u_test")
	sess.Set(totpEnrollmentIDKey, "e1")
	sess.Set(totpEnrollmentSubjectKey, "u_test")
	sess.Set(totpEnrollmentStartedKey, time.Now().Unix())
	sessionID := sess.ID()
	testza.AssertNoError(t, auth.Authenticate(sess))
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sessionID)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusServiceUnavailable, ctx.Response().StatusCode())
}

func TestTOTPEnrollConfirmAPI_RejectsUnboundEnrollment(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	testza.AssertNoError(t, config.Initialize(testLogger()))

	store := setupTestStore()
	handler := TOTPEnrollConfirmAPI(store)
	ctx, app := createTestContext("POST", "/totp/enroll/confirm", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "enroll_id=attacker-enrollment&code=123456")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")
	sessionID := sess.ID()
	testza.AssertNoError(t, auth.Authenticate(sess))
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sessionID)

	testza.AssertNoError(t, handler(ctx))
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
	testza.AssertContains(t, string(ctx.Response().Body()), "invalid_enrollment")
}

func TestTOTPEnrollConfirmAPIReturnsBackupCodesWhenSessionCleanupFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/totp/enroll/confirm" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(herald.TOTPEnrollConfirmResponse{
			Subject:     "u_test",
			TotpEnabled: true,
			BackupCodes: []string{"backup-1", "backup-2"},
		})
	}))
	defer server.Close()

	ResetHeraldClientForTest()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_URL", server.URL)
	t.Setenv("HERALD_TOTP_ENABLED", "true")
	testza.AssertNoError(t, config.Initialize(testLogger()))
	testza.AssertNoError(t, InitHeraldClient(testLogger()))

	store := setupTestStore()
	failingStorage := &failSetStorage{Storage: store.Storage}
	store.Storage = failingStorage
	ctx, app := createTestContext("POST", "/totp/enroll/confirm", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "enroll_id=e1&code=123456")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")
	sess.Set(totpEnrollmentIDKey, "e1")
	sess.Set(totpEnrollmentSubjectKey, "u_test")
	sess.Set(totpEnrollmentStartedKey, time.Now().Unix())
	testza.AssertNoError(t, auth.Authenticate(sess))
	testza.AssertNoError(t, sess.Save())
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sess.ID())
	failingStorage.fail = true

	testza.AssertNoError(t, TOTPEnrollConfirmAPI(store)(ctx))
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertContains(t, string(ctx.Response().Body()), "backup-1")
	testza.AssertContains(t, string(ctx.Response().Body()), "backup-2")
	testza.AssertNotContains(t, string(ctx.Response().Body()), "session_save_failed")
}

func TestTOTPEnrollRequiresRecentAuthentication(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	testza.AssertNoError(t, config.Initialize(testLogger()))

	store := setupTestStore()
	ctx, app := createTestContext("POST", "/totp/enroll", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "u_test")
	sessionID := sess.ID()
	testza.AssertNoError(t, auth.Authenticate(sess))
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sessionID)
	sess, err = store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("created_at", time.Now().Add(-totpEnrollmentValidity-time.Minute).Unix())
	testza.AssertNoError(t, sess.Save())

	testza.AssertNoError(t, TOTPEnrollRoute(store)(ctx))
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}
