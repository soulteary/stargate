package handlers

import (
	"errors"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/config"
)

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

// Ensure *session.Store satisfies SessionStoreForCheck at compile time.
var _ SessionStoreForCheck = (*session.Store)(nil)

// Note: CheckRoute branches for forwardauth.ErrStepUpRequired and forwardauth.ErrSessionRequired
// are not unit-tested here (would require mocking forwardauth.Handler). They are covered by
// integration tests when step-up or session-required paths are triggered.
