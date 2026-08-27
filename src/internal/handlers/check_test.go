package handlers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
)

func TestTrustedHeaderAuthResultPropagatesCancellation(t *testing.T) {
	originalLookup := lookupTrustedHeaderUser
	t.Cleanup(func() { lookupTrustedHeaderUser = originalLookup })

	requestCtx, cancel := context.WithCancel(context.Background())
	cancel()
	lookupTrustedHeaderUser = func(ctx context.Context, phone, mail string) *warden.AllowListUser {
		select {
		case <-ctx.Done():
		default:
			t.Fatal("Warden lookup did not receive the canceled request context")
		}
		return &warden.AllowListUser{UserID: "user1", Phone: phone, Mail: mail}
	}

	result, err := trustedHeaderAuthResult(requestCtx, "13800138000", "")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "user1", result.UserID)
}

// TestCheckRoute_HandlerNil verifies that when GetForwardAuthHandler returns nil,
// the check handler returns 500 and "ForwardAuth handler not initialized".
func TestCheckRoute_HandlerNil(t *testing.T) {
	save := forwardAuthHandler
	defer func() { forwardAuthHandler = save }()
	forwardAuthHandler = nil

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json"}, "")
	defer app.ReleaseCtx(ctx)

	err := handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusInternalServerError, ctx.Response().StatusCode())
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, "ForwardAuth handler not initialized")
}

// mockSessionStoreFailing is a SessionStoreForCheck that always returns an error from Get.
type mockSessionStoreFailing struct{}

func (m *mockSessionStoreFailing) Get(_ fiber.Ctx) (*session.Session, error) {
	return nil, errors.New("store error")
}

// TestCheckRoute_SessionStoreError verifies that when store.Get returns an error,
// the check handler returns 500 and session store failed message.
func TestCheckRoute_SessionStoreError(t *testing.T) {
	setupCheckHeaderConfig(t)

	handler := CheckRoute(&mockSessionStoreFailing{})

	ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json"}, "")
	defer app.ReleaseCtx(ctx)

	err := handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusInternalServerError, ctx.Response().StatusCode())
	body := string(ctx.Response().Body())
	testza.AssertTrue(t, len(body) > 0)
	// Response may be JSON with translated message (e.g. "failed to access session store")
	testza.AssertTrue(t, strings.Contains(body, "session_store_failed") || strings.Contains(body, "session store"), "body should indicate session store failure: %s", body)
}

func TestCheckRouteIgnoresForwardedRedirectHeadersFromUntrustedPeer(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("CALLBACK_ALLOWED_HOSTS", "app.example.com")
	t.Setenv("TRUSTED_PROXIES", "")
	testLogger := testLoggerCheckHeaders()
	testza.AssertNoError(t, config.Initialize(testLogger))
	InitForwardAuthHandler(testLogger)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Accept":            "text/html",
		"Host":              "auth.example.com",
		"X-Forwarded-Host":  "evil.example.net",
		"X-Forwarded-Proto": "https",
	}, "")
	defer app.ReleaseCtx(ctx)

	err := CheckRoute(setupTestStore())(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusFound, ctx.Response().StatusCode())
	location := string(ctx.Response().Header.Peek("Location"))
	testza.AssertEqual(t, "http://auth.example.com/_login?callback=auth.example.com", location)
	testza.AssertNotContains(t, location, "evil.example.net")
}

func TestCheckRouteUsesForwardedRedirectHeadersFromTrustedPeer(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("CALLBACK_ALLOWED_HOSTS", "app.example.com")
	t.Setenv("TRUSTED_PROXIES", "0.0.0.0")
	testLogger := testLoggerCheckHeaders()
	testza.AssertNoError(t, config.Initialize(testLogger))
	InitForwardAuthHandler(testLogger)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Accept":            "text/html",
		"Host":              "auth.example.com",
		"X-Forwarded-Host":  "app.example.com",
		"X-Forwarded-Proto": "https",
	}, "")
	defer app.ReleaseCtx(ctx)

	err := CheckRoute(setupTestStore())(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusFound, ctx.Response().StatusCode())
	testza.AssertEqual(t, "https://auth.example.com/_login?callback=app.example.com", string(ctx.Response().Header.Peek("Location")))
}

func TestCheckRouteRejectsStatelessPasswordOnStepUpPath(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("PASSWORD_HEADER_AUTH_ENABLED", "true")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	logger := testLoggerCheckHeaders()
	testza.AssertNoError(t, config.Initialize(logger))
	InitForwardAuthHandler(logger)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Accept":            "application/json",
		"Stargate-Password": "test123",
		"X-Forwarded-Uri":   "/admin/settings",
	}, "")
	defer app.ReleaseCtx(ctx)

	testza.AssertNoError(t, CheckRoute(setupTestStore())(ctx))
	testza.AssertEqual(t, fiber.StatusForbidden, ctx.Response().StatusCode())
	testza.AssertContains(t, string(ctx.Response().Body()), "step-up requires an authenticated session")
}

func TestCheckRouteRejectsStatelessIdentityUsingVerifiedSession(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("PASSWORD_HEADER_AUTH_ENABLED", "true")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	logger := testLoggerCheckHeaders()
	testza.AssertNoError(t, config.Initialize(logger))
	InitForwardAuthHandler(logger)

	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Accept":            "application/json",
		"Stargate-Password": "test123",
		"X-Forwarded-Uri":   "/admin/settings",
	}, "")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "session-user")
	sess.Set("step_up_verified_at", time.Now().Unix())
	testza.AssertNoError(t, auth.Authenticate(sess))
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sess.ID())

	testza.AssertNoError(t, CheckRoute(store)(ctx))
	testza.AssertEqual(t, fiber.StatusForbidden, ctx.Response().StatusCode())
	testza.AssertContains(t, string(ctx.Response().Body()), "does not match the authentication identity")
}

func TestCheckRouteAllowsVerifiedSessionOnStepUpPath(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	logger := testLoggerCheckHeaders()
	testza.AssertNoError(t, config.Initialize(logger))
	InitForwardAuthHandler(logger)

	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Accept":          "application/json",
		"X-Forwarded-Uri": "/admin/settings",
	}, "")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_id", "session-user")
	sess.Set("step_up_verified_at", time.Now().Unix())
	testza.AssertNoError(t, auth.Authenticate(sess))
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sess.ID())

	testza.AssertNoError(t, CheckRoute(store)(ctx))
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "session-user", string(ctx.Response().Header.Peek("X-Forwarded-User")))
}

// Ensure *session.Store satisfies SessionStoreForCheck at compile time.
var _ SessionStoreForCheck = (*session.Store)(nil)

// Note: CheckRoute branches for forwardauth.ErrStepUpRequired and forwardauth.ErrSessionRequired
// are not unit-tested here (would require mocking forwardauth.Handler). They are covered by
// integration tests when step-up or session-required paths are triggered.
