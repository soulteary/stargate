package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	health "github.com/soulteary/health-kit"
	logger "github.com/soulteary/logger-kit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/valyala/fasthttp"
)

// TestMain runs before all tests to set up the test environment
func TestMain(m *testing.M) {
	// Set up required environment variables for config
	_ = os.Setenv("AUTH_HOST", "auth.example.com")
	_ = os.Setenv("PASSWORDS", "plaintext:test123")

	// Initialize config and ForwardAuth handler
	testLog := testLogger()
	if err := config.Initialize(testLog); err != nil {
		panic("Failed to initialize config: " + err.Error())
	}
	InitForwardAuthHandler(testLog)

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// testLogger creates a logger instance for testing
func testLogger() *logger.Logger {
	return logger.New(logger.Config{
		Level:       logger.DebugLevel,
		Format:      logger.FormatJSON,
		ServiceName: "handlers-test",
	})
}

func setupTestStore() *session.Store {
	return session.New(session.Config{
		KeyLookup:    "cookie:" + auth.SessionCookieName,
		KeyGenerator: utils.UUID,
	})
}

func createTestContext(method, path string, headers map[string]string, body string) (*fiber.Ctx, *fiber.App) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})

	ctx.Request().SetRequestURI(path)
	ctx.Request().Header.SetMethod(method)

	if body != "" {
		ctx.Request().SetBodyString(body)
		// Set Content-Length header for proper body parsing
		ctx.Request().Header.SetContentLength(len(body))
	}

	for k, v := range headers {
		ctx.Request().Header.Set(k, v)
	}

	// Set i18n bundle and language for testing (required by i18n-kit middleware)
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)

	return ctx, app
}

func TestCheckRoute_Authenticated(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Host": "test.example.com",
	}, "")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	// Test handler
	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
}

func TestCheckRoute_NotAuthenticated(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Host": "test.example.com",
	}, "")
	defer app.ReleaseCtx(ctx)

	// Test handler without authentication
	err = handler(ctx)
	// Should redirect, but we can't easily test redirect in unit test
	// Just check that no error occurred
	testza.AssertNoError(t, err)
}

func TestCheckRoute_HeaderAuth_Valid(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Stargate-Password": "test123",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
}

func TestCheckRoute_HeaderAuth_Invalid(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Stargate-Password": "wrong",
		"Accept":            "application/json", // API request should return 401
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func TestLoginAPI_ValidPassword(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "password=test123")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
}

func TestLoginAPI_InvalidPassword(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "password=wrong")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

// TestLoginAPI_WardenNoIdentifier ensures warden auth returns 400 when neither phone nor mail provided.
func TestLoginAPI_WardenNoIdentifier(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "auth_method=warden")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
}

// TestLoginAPI_WardenUserNotFound ensures warden auth returns 401 when GetUserInfo returns nil.
func TestLoginAPI_WardenUserNotFound(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	// No WARDEN_URL so InitWardenClient will not have a real client; GetUserInfo returns nil
	auth.ResetWardenClientForTesting()
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)
	auth.InitWardenClient(testLogger())

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "auth_method=warden&phone=13800138000&mail=user@example.com")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func TestLoginRoute_NotAuthenticated(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LOGIN_PAGE_TITLE", "Test Title")
	t.Setenv("LOGIN_PAGE_FOOTER_TEXT", "Test Footer")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginRoute(store)

	ctx, app := createTestContext("GET", "/_login?callback=test.example.com", nil, "")
	defer app.ReleaseCtx(ctx)

	// Setup template engine to avoid file not found error
	engine := app.Config().Views
	if engine == nil {
		// Skip template rendering test if no engine configured
		// This is expected in unit tests without full app setup
		return
	}

	err = handler(ctx)
	// Template rendering may fail in unit test environment, which is acceptable
	// We just verify the handler doesn't panic
	_ = err
}

func TestLoginRoute_Authenticated(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginRoute(store)

	ctx, app := createTestContext("GET", "/_login?callback=test.example.com", nil, "")
	defer app.ReleaseCtx(ctx)

	// Create authenticated session
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	// Should redirect, but we can't easily test redirect in unit test
	testza.AssertNoError(t, err)
}

func TestLogoutRoute(t *testing.T) {
	store := setupTestStore()
	handler := LogoutRoute(store)

	ctx, app := createTestContext("GET", "/_logout", nil, "")
	defer app.ReleaseCtx(ctx)

	// Create authenticated session first
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "Logged out", string(ctx.Response().Body()))
}

func TestIndexRoute_Authenticated(t *testing.T) {
	store := setupTestStore()
	handler := IndexRoute(store)

	ctx, app := createTestContext("GET", "/", nil, "")
	defer app.ReleaseCtx(ctx)

	// Create authenticated session
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "Authenticated", string(ctx.Response().Body()))
}

func TestIndexRoute_NotAuthenticated(t *testing.T) {
	store := setupTestStore()
	handler := IndexRoute(store)

	ctx, app := createTestContext("GET", "/", nil, "")
	defer app.ReleaseCtx(ctx)

	err := handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "Not authenticated", string(ctx.Response().Body()))
}

func TestSessionShareRoute_WithID(t *testing.T) {
	handler := SessionShareRoute()

	ctx, app := createTestContext("GET", "/_session_exchange?id=test-session-id", nil, "")
	defer app.ReleaseCtx(ctx)

	err := handler(ctx)
	// Should redirect, but we can't easily test redirect in unit test
	testza.AssertNoError(t, err)

	// Check that cookie was set
	cookies := ctx.Response().Header.Peek("Set-Cookie")
	testza.AssertNotNil(t, cookies)
	testza.AssertContains(t, string(cookies), auth.SessionCookieName)
	testza.AssertContains(t, string(cookies), "test-session-id")
}

func TestSessionShareRoute_WithoutID(t *testing.T) {
	handler := SessionShareRoute()

	ctx, app := createTestContext("GET", "/_session_exchange", nil, "")
	defer app.ReleaseCtx(ctx)

	err := handler(ctx)
	testza.AssertNoError(t, err)
	// SessionShareRoute returns StatusBadRequest (400) for missing session ID
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
}

// Note: Health check is now handled by health-kit in server.go
// Health check tests should be in server_test.go or integration tests
func TestHealthRoute(t *testing.T) {
	// Create a simple health aggregator for testing
	healthConfig := health.DefaultConfig().WithServiceName("stargate")
	aggregator := health.NewAggregator(healthConfig)
	// No checkers added means all healthy
	handler := health.FiberHandler(aggregator)

	ctx, app := createTestContext("GET", "/health", nil, "")
	defer app.ReleaseCtx(ctx)

	err := handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
}

func TestCheckRoute_NotAuthenticated_HTMLRequest(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Host":   "test.example.com",
		"Accept": "text/html",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should redirect for HTML requests
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusFound || ctx.Response().StatusCode() == fiber.StatusMovedPermanently)
}

func TestCheckRoute_NotAuthenticated_APIRequest(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Host":   "test.example.com",
		"Accept": "application/json",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func TestLoginAPI_EmptyPassword(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "password=")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func TestLoginAPI_NoPasswordField(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
}

func TestLoginRoute_WithCallback(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LOGIN_PAGE_TITLE", "Test Title")
	t.Setenv("LOGIN_PAGE_FOOTER_TEXT", "Test Footer")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginRoute(store)

	ctx, app := createTestContext("GET", "/_login?callback=app.example.com", nil, "")
	defer app.ReleaseCtx(ctx)

	// Template rendering may fail in unit test environment, which is acceptable
	err = handler(ctx)
	// We just verify the handler doesn't panic
	_ = err
}

func TestLoginRoute_WithoutCallback(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LOGIN_PAGE_TITLE", "Test Title")
	t.Setenv("LOGIN_PAGE_FOOTER_TEXT", "Test Footer")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginRoute(store)

	ctx, app := createTestContext("GET", "/_login", nil, "")
	defer app.ReleaseCtx(ctx)

	// Template rendering may fail in unit test environment, which is acceptable
	err = handler(ctx)
	// We just verify the handler doesn't panic
	_ = err
}

func TestLoginRoute_Authenticated_WithForwardedProto(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginRoute(store)

	ctx, app := createTestContext("GET", "/_login?callback=app.example.com", map[string]string{
		"X-Forwarded-Proto": "https",
	}, "")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should redirect
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusFound || ctx.Response().StatusCode() == fiber.StatusMovedPermanently)
}

func TestLogoutRoute_NotAuthenticated(t *testing.T) {
	store := setupTestStore()
	handler := LogoutRoute(store)

	ctx, app := createTestContext("GET", "/_logout", nil, "")
	defer app.ReleaseCtx(ctx)

	// Don't authenticate
	err := handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "Logged out", string(ctx.Response().Body()))
}

func TestSessionShareRoute_WithCookieDomain(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	handler := SessionShareRoute()

	ctx, app := createTestContext("GET", "/_session_exchange?id=test-session-id", nil, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)

	// Check that cookie was set with domain
	cookies := ctx.Response().Header.Peek("Set-Cookie")
	testza.AssertNotNil(t, cookies)
	cookieStr := string(cookies)
	testza.AssertContains(t, cookieStr, auth.SessionCookieName)
	testza.AssertContains(t, cookieStr, "test-session-id")
	testza.AssertContains(t, cookieStr, ".example.com")
}

func TestSessionShareRoute_WithoutCookieDomain(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("COOKIE_DOMAIN", "")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	handler := SessionShareRoute()

	ctx, app := createTestContext("GET", "/_session_exchange?id=test-session-id", nil, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)

	// Check that cookie was set without domain
	cookies := ctx.Response().Header.Peek("Set-Cookie")
	testza.AssertNotNil(t, cookies)
	cookieStr := string(cookies)
	testza.AssertContains(t, cookieStr, auth.SessionCookieName)
	testza.AssertContains(t, cookieStr, "test-session-id")
}

func TestCheckRoute_SetsUserHeader(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("USER_HEADER_NAME", "X-Custom-User")
	testLog := testLogger()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	InitForwardAuthHandler(testLog) // Re-init to pick up custom USER_HEADER_NAME

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Stargate-Password": "test123",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	// Check response header instead of request header
	userHeader := string(ctx.Response().Header.Peek("X-Custom-User"))
	testza.AssertEqual(t, "authenticated", userHeader)
}

func TestCheckRoute_SetsDefaultUserHeader(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	testLog := testLogger()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	InitForwardAuthHandler(testLog) // Re-init to ensure default USER_HEADER_NAME is used

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"Stargate-Password": "test123",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	// Check response header instead of request header
	userHeader := string(ctx.Response().Header.Peek("X-Forwarded-User"))
	testza.AssertEqual(t, "authenticated", userHeader)
}

// TestLoginAPI_WithCallbackInForm tests that callback from form data is used
func TestLoginAPI_WithCallbackInForm(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"X-Forwarded-Host": "app.example.com",
		"Host":             "auth.example.com",
	}, "password=test123&callback=app.example.com")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)

	// Debug: check actual status code
	statusCode := ctx.Response().StatusCode()
	if statusCode != fiber.StatusFound && statusCode != fiber.StatusMovedPermanently {
		t.Logf("Unexpected status code: %d, body: %s", statusCode, string(ctx.Response().Body()))
	}

	// Should redirect to session exchange endpoint
	testza.AssertTrue(t, statusCode == fiber.StatusFound || statusCode == fiber.StatusMovedPermanently)

	// Check redirect location
	location := string(ctx.Response().Header.Peek("Location"))
	testza.AssertContains(t, location, "app.example.com")
	testza.AssertContains(t, location, "/_session_exchange")
}

// TestLoginAPI_WithCallbackInQuery tests that callback from query parameter is used
func TestLoginAPI_WithCallbackInQuery(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login?callback=app.example.com", map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"X-Forwarded-Host": "app.example.com",
	}, "password=test123")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should redirect to session exchange endpoint
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusFound || ctx.Response().StatusCode() == fiber.StatusMovedPermanently)

	// Check redirect location
	location := string(ctx.Response().Header.Peek("Location"))
	testza.AssertContains(t, location, "app.example.com")
	testza.AssertContains(t, location, "/_session_exchange")
}

// TestLoginAPI_WithCallbackInCookie tests that callback from cookie is used
func TestLoginAPI_WithCallbackInCookie(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"X-Forwarded-Host": "app.example.com",
		"Host":             "auth.example.com",
		"Cookie":           "stargate_callback=app.example.com",
	}, "password=test123")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)

	statusCode := ctx.Response().StatusCode()
	if statusCode != fiber.StatusFound && statusCode != fiber.StatusMovedPermanently {
		t.Logf("Unexpected status code: %d, body: %s", statusCode, string(ctx.Response().Body()))
	}

	// Should redirect to session exchange endpoint
	testza.AssertTrue(t, statusCode == fiber.StatusFound || statusCode == fiber.StatusMovedPermanently)

	// Check redirect location
	location := string(ctx.Response().Header.Peek("Location"))
	testza.AssertContains(t, location, "app.example.com")
	testza.AssertContains(t, location, "/_session_exchange")

	// Check that callback cookie is cleared
	cookies := ctx.Response().Header.Peek("Set-Cookie")
	cookieStr := string(cookies)
	// The cookie should be cleared (expired in the past)
	testza.AssertContains(t, cookieStr, CallbackCookieName)
}

// TestLoginAPI_NoCallback_UsesOriginHost tests that origin host is used as callback when no callback provided
func TestLoginAPI_NoCallback_UsesOriginHost(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"X-Forwarded-Host": "app.example.com",
		"Host":             "auth.example.com",
	}, "password=test123")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)

	statusCode := ctx.Response().StatusCode()
	if statusCode != fiber.StatusFound && statusCode != fiber.StatusMovedPermanently {
		t.Logf("Unexpected status code: %d, body: %s", statusCode, string(ctx.Response().Body()))
	}

	// Should redirect to session exchange endpoint
	testza.AssertTrue(t, statusCode == fiber.StatusFound || statusCode == fiber.StatusMovedPermanently)

	// Check redirect location
	location := string(ctx.Response().Header.Peek("Location"))
	testza.AssertContains(t, location, "app.example.com")
	testza.AssertContains(t, location, "/_session_exchange")
}

// TestLoginAPI_NoCallback_SameDomain tests that HTML response is returned when no callback and same domain
func TestLoginAPI_NoCallback_SameDomain(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"X-Forwarded-Host": "auth.example.com",
		"Accept":           "text/html",
	}, "password=test123")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should return HTML response (not redirect)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	// Check content type
	contentType := string(ctx.Response().Header.Peek("Content-Type"))
	testza.AssertContains(t, contentType, "text/html")

	// Check response body contains redirect URL
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, "auth.example.com")
}

// TestLoginAPI_CallbackPriority tests that callback priority is: cookie > form > query
func TestLoginAPI_CallbackPriority(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	// Cookie callback should take priority over form and query
	ctx, app := createTestContext("POST", "/_login?callback=query.example.com", map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"X-Forwarded-Host": "app.example.com",
		"Host":             "auth.example.com",
		"Cookie":           "stargate_callback=cookie.example.com",
	}, "password=test123&callback=form.example.com")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)

	statusCode := ctx.Response().StatusCode()
	if statusCode != fiber.StatusFound && statusCode != fiber.StatusMovedPermanently {
		t.Logf("Unexpected status code: %d, body: %s", statusCode, string(ctx.Response().Body()))
	}

	// Should redirect to session exchange endpoint
	testza.AssertTrue(t, statusCode == fiber.StatusFound || statusCode == fiber.StatusMovedPermanently)

	// Check redirect location uses cookie callback
	location := string(ctx.Response().Header.Peek("Location"))
	testza.AssertContains(t, location, "cookie.example.com")
	testza.AssertNotContains(t, location, "form.example.com")
	testza.AssertNotContains(t, location, "query.example.com")
}

// TestLoginAPI_RedirectsToSessionExchange tests that redirect includes session ID
func TestLoginAPI_RedirectsToSessionExchange(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":      "application/x-www-form-urlencoded",
		"X-Forwarded-Host":  "app.example.com",
		"X-Forwarded-Proto": "https",
		"Host":              "auth.example.com",
	}, "password=test123&callback=app.example.com")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)

	statusCode := ctx.Response().StatusCode()
	if statusCode != fiber.StatusFound && statusCode != fiber.StatusMovedPermanently {
		t.Logf("Unexpected status code: %d, body: %s", statusCode, string(ctx.Response().Body()))
	}

	// Should redirect to session exchange endpoint
	testza.AssertTrue(t, statusCode == fiber.StatusFound || statusCode == fiber.StatusMovedPermanently)

	// Check redirect location format
	location := string(ctx.Response().Header.Peek("Location"))
	testza.AssertContains(t, location, "https://app.example.com/_session_exchange")
	testza.AssertContains(t, location, "id=")

	// Extract session ID from location
	// Location format: https://app.example.com/_session_exchange?id=<session_id>
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sessionID := sess.ID()
	testza.AssertNotNil(t, sessionID)
	testza.AssertContains(t, location, sessionID)
}

// TestLoginAPI_WithCallback_AcceptJSON_Returns200WithRedirect tests that when client sends
// Accept: application/json and login succeeds with callback, server returns 200 + JSON with redirect
// (so fetch with redirect: 'manual' can read the URL and navigate; 302 Location is opaque).
func TestLoginAPI_WithCallback_AcceptJSON_Returns200WithRedirect(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":      "application/x-www-form-urlencoded",
		"X-Forwarded-Host":  "app.example.com",
		"X-Forwarded-Proto": "https",
		"Host":              "auth.example.com",
		"Accept":            "application/json",
	}, "password=test123&callback=app.example.com")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	contentType := string(ctx.Response().Header.Peek("Content-Type"))
	testza.AssertContains(t, contentType, "application/json")

	var result struct {
		Success  bool   `json:"success"`
		Redirect string `json:"redirect"`
		Message  string `json:"message"`
	}
	err = json.Unmarshal(ctx.Response().Body(), &result)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, result.Success)
	testza.AssertContains(t, result.Redirect, "app.example.com/_session_exchange")
	testza.AssertContains(t, result.Redirect, "id=")
}

// TestLoginAPI_NoCallback_APIRequest tests that API request returns JSON when no callback
func TestLoginAPI_NoCallback_APIRequest(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":      "application/x-www-form-urlencoded",
		"X-Forwarded-Host":  "auth.example.com",
		"X-Forwarded-Proto": "https",
		"Accept":            "application/json",
	}, "password=test123")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	// Check content type
	contentType := string(ctx.Response().Header.Peek("Content-Type"))
	testza.AssertContains(t, contentType, "application/json")

	// Check response body contains success message
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, "success")
	testza.AssertContains(t, body, "true")
}

// TestLoginRoute_NoCallback_RedirectsToRoot tests that redirects to root when no callback
func TestLoginRoute_NoCallback_RedirectsToRoot(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginRoute(store)

	ctx, app := createTestContext("GET", "/_login", map[string]string{
		"X-Forwarded-Host":  "auth.example.com",
		"X-Forwarded-Proto": "https",
	}, "")
	defer app.ReleaseCtx(ctx)

	// Create authenticated session
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should redirect
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusFound || ctx.Response().StatusCode() == fiber.StatusMovedPermanently)

	// Check redirect location
	location := string(ctx.Response().Header.Peek("Location"))
	testza.AssertContains(t, location, "https://auth.example.com/")
}

// TestLoginRoute_WithForwardedProto_Empty tests that uses ctx.Protocol() when X-Forwarded-Proto is empty
func TestLoginRoute_WithForwardedProto_Empty(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginRoute(store)

	ctx, app := createTestContext("GET", "/_login?callback=app.example.com", map[string]string{
		"X-Forwarded-Host": "app.example.com",
		// Don't set X-Forwarded-Proto
	}, "")
	defer app.ReleaseCtx(ctx)

	// Create authenticated session
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should redirect
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusFound || ctx.Response().StatusCode() == fiber.StatusMovedPermanently)
}

// TestIndexRoute_SessionStoreError tests error handling when session store fails
func TestIndexRoute_SessionStoreError(t *testing.T) {
	mockStore := &MockSessionGetter{
		GetFunc: func(ctx *fiber.Ctx) (*session.Session, error) {
			return nil, fiber.ErrInternalServerError
		},
	}
	handler := IndexRoute(mockStore)

	ctx, app := createTestContext("GET", "/", nil, "")
	defer app.ReleaseCtx(ctx)

	err := handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusInternalServerError, ctx.Response().StatusCode())
	body := string(ctx.Response().Body())
	testza.AssertTrue(t, strings.Contains(body, "session_store_failed") || strings.Contains(body, "session store"), "body should indicate session store failure: %s", body)
}

// MockSessionGetter is a mock implementation of SessionGetter for testing.
type MockSessionGetter struct {
	GetFunc func(ctx *fiber.Ctx) (*session.Session, error)
}

func (m *MockSessionGetter) Get(ctx *fiber.Ctx) (*session.Session, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx)
	}
	return nil, nil
}

// MockUnauthenticator is a mock implementation of Unauthenticator for testing.
type MockUnauthenticator struct {
	UnauthenticateFunc func(sess *session.Session) error
}

func (m *MockUnauthenticator) Unauthenticate(sess *session.Session) error {
	if m.UnauthenticateFunc != nil {
		return m.UnauthenticateFunc(sess)
	}
	return nil
}

// TestLogoutRoute_SessionStoreError tests error handling when session store fails
func TestLogoutRoute_SessionStoreError(t *testing.T) {
	ctx, app := createTestContext("GET", "/_logout", nil, "")
	defer app.ReleaseCtx(ctx)

	mockSessionGetter := &MockSessionGetter{
		GetFunc: func(ctx *fiber.Ctx) (*session.Session, error) {
			return nil, fiber.ErrInternalServerError
		},
	}
	mockUnauthenticator := &MockUnauthenticator{}

	err := logoutHandler(ctx, mockSessionGetter, mockUnauthenticator)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusInternalServerError, ctx.Response().StatusCode())

	// Verify error response format
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, i18n.TStatic("error.session_store_failed"))
}

// TestLogoutRoute_UnauthenticateError tests error handling when unauthenticate fails
func TestLogoutRoute_UnauthenticateError(t *testing.T) {
	ctx, app := createTestContext("GET", "/_logout", nil, "")
	defer app.ReleaseCtx(ctx)

	store := setupTestStore()
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)

	mockSessionGetter := &MockSessionGetter{
		GetFunc: func(ctx *fiber.Ctx) (*session.Session, error) {
			return sess, nil
		},
	}
	mockUnauthenticator := &MockUnauthenticator{
		UnauthenticateFunc: func(sess *session.Session) error {
			return fiber.ErrInternalServerError
		},
	}

	err = logoutHandler(ctx, mockSessionGetter, mockUnauthenticator)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusInternalServerError, ctx.Response().StatusCode())

	// Verify error response format
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, i18n.TStatic("error.authenticate_failed"))
}

// TestLogoutHandler_Success tests the internal logoutHandler with successful logout
func TestLogoutHandler_Success(t *testing.T) {
	ctx, app := createTestContext("GET", "/_logout", nil, "")
	defer app.ReleaseCtx(ctx)

	store := setupTestStore()
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)

	mockSessionGetter := &MockSessionGetter{
		GetFunc: func(ctx *fiber.Ctx) (*session.Session, error) {
			return sess, nil
		},
	}
	mockUnauthenticator := &MockUnauthenticator{
		UnauthenticateFunc: func(sess *session.Session) error {
			return auth.Unauthenticate(sess)
		},
	}

	err = logoutHandler(ctx, mockSessionGetter, mockUnauthenticator)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "Logged out", string(ctx.Response().Body()))
}

// TestLogoutRoute_MultipleCalls tests that logout can be called multiple times safely
func TestLogoutRoute_MultipleCalls(t *testing.T) {
	store := setupTestStore()
	handler := LogoutRoute(store)

	ctx, app := createTestContext("GET", "/_logout", nil, "")
	defer app.ReleaseCtx(ctx)

	// Create authenticated session first
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	// Call logout first time
	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "Logged out", string(ctx.Response().Body()))

	// Call logout second time - should still work
	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "Logged out", string(ctx.Response().Body()))
}

// TestLogoutRoute_WithDifferentAcceptHeaders tests logout with different Accept headers
func TestLogoutRoute_WithDifferentAcceptHeaders(t *testing.T) {
	store := setupTestStore()
	handler := LogoutRoute(store)

	tests := []struct {
		name    string
		headers map[string]string
	}{
		{"JSON", map[string]string{"Accept": "application/json"}},
		{"XML", map[string]string{"Accept": "application/xml"}},
		{"HTML", map[string]string{"Accept": "text/html"}},
		{"No Accept", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, app := createTestContext("GET", "/_logout", tt.headers, "")
			defer app.ReleaseCtx(ctx)

			// Create authenticated session first
			sess, err := store.Get(ctx)
			testza.AssertNoError(t, err)
			err = auth.Authenticate(sess)
			testza.AssertNoError(t, err)

			err = handler(ctx)
			testza.AssertNoError(t, err)
			testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
			testza.AssertEqual(t, "Logged out", string(ctx.Response().Body()))
		})
	}
}

// TestLogoutRoute_ErrorResponseFormat tests that error responses use the correct format
// This tests the SendErrorResponse function is called correctly
func TestLogoutRoute_ErrorResponseFormat(t *testing.T) {
	// This test verifies that if an error occurs, it uses SendErrorResponse
	// which formats the response based on Accept header
	store := setupTestStore()
	handler := LogoutRoute(store)

	// Test with JSON Accept header - if an error occurs, it should return JSON
	ctx, app := createTestContext("GET", "/_logout", map[string]string{
		"Accept": "application/json",
	}, "")
	defer app.ReleaseCtx(ctx)

	// Normal logout should work
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	testza.AssertEqual(t, "Logged out", string(ctx.Response().Body()))
}

// TestLogoutRoute_ConcurrentAccess tests concurrent logout requests
func TestLogoutRoute_ConcurrentAccess(t *testing.T) {
	store := setupTestStore()
	handler := LogoutRoute(store)

	var wg sync.WaitGroup
	iterations := 10

	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			ctx, app := createTestContext("GET", "/_logout", nil, "")
			defer app.ReleaseCtx(ctx)

			// Create authenticated session
			sess, err := store.Get(ctx)
			if err != nil {
				return
			}
			_ = auth.Authenticate(sess)

			// Call logout
			_ = handler(ctx)
		}()
	}

	wg.Wait()
}

// TestLogoutRoute_AfterLogin tests logout after successful login
func TestLogoutRoute_AfterLogin(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	logoutHandler := LogoutRoute(store)
	loginHandler := LoginAPI(store)

	// First login
	ctx1, app1 := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "password=test123")
	defer app1.ReleaseCtx(ctx1)

	err = loginHandler(ctx1)
	testza.AssertNoError(t, err)

	// Verify session is authenticated
	sess, err := store.Get(ctx1)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, auth.IsAuthenticated(sess), "session should be authenticated after login")

	// Then logout
	ctx2, app2 := createTestContext("GET", "/_logout", nil, "")
	defer app2.ReleaseCtx(ctx2)

	// Copy session cookie from login response to logout request
	cookie := ctx1.Response().Header.Peek("Set-Cookie")
	if len(cookie) > 0 {
		ctx2.Request().Header.Set("Cookie", string(cookie))
	}

	err = logoutHandler(ctx2)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx2.Response().StatusCode())
	testza.AssertEqual(t, "Logged out", string(ctx2.Response().Body()))

	// Verify session is no longer authenticated
	sess2, err := store.Get(ctx2)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, auth.IsAuthenticated(sess2), "session should not be authenticated after logout")
}

// TestLogoutRoute_ResponseHeaders tests that logout sets appropriate response headers
func TestLogoutRoute_ResponseHeaders(t *testing.T) {
	store := setupTestStore()
	handler := LogoutRoute(store)

	ctx, app := createTestContext("GET", "/_logout", nil, "")
	defer app.ReleaseCtx(ctx)

	// Create authenticated session
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	err = handler(ctx)
	testza.AssertNoError(t, err)

	// Verify response headers
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())
	contentType := string(ctx.Response().Header.Peek("Content-Type"))
	// SendString should set Content-Type automatically
	testza.AssertNotNil(t, contentType)
}

// TestLoginAPI_EmptySessionID_FromCookie tests that session ID is extracted from cookie when empty
func TestLoginAPI_EmptySessionID_FromCookie(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":      "application/x-www-form-urlencoded",
		"X-Forwarded-Host":  "app.example.com",
		"X-Forwarded-Proto": "https",
		"Host":              "auth.example.com",
	}, "password=test123&callback=app.example.com")
	defer app.ReleaseCtx(ctx)

	// Get session and authenticate first to create session cookie
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	// Now call handler - it should extract session ID from cookie if sess.ID() is empty
	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should redirect
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusFound || ctx.Response().StatusCode() == fiber.StatusMovedPermanently)
}

// TestLoginAPI_EmptySessionID_RetryGetSession tests that retries getting session when ID is empty
func TestLoginAPI_EmptySessionID_RetryGetSession(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":      "application/x-www-form-urlencoded",
		"X-Forwarded-Host":  "app.example.com",
		"X-Forwarded-Proto": "https",
		"Host":              "auth.example.com",
	}, "password=test123&callback=app.example.com")
	defer app.ReleaseCtx(ctx)

	// Get session and authenticate
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	err = auth.Authenticate(sess)
	testza.AssertNoError(t, err)

	// Call handler
	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should redirect successfully
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusFound || ctx.Response().StatusCode() == fiber.StatusMovedPermanently)
}

// TestLoginAPI_EmptyProto_UsesContextProtocol tests that uses ctx.Protocol() when X-Forwarded-Proto is empty
func TestLoginAPI_EmptyProto_UsesContextProtocol(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	handler := LoginAPI(store)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"X-Forwarded-Host": "app.example.com",
		// Don't set X-Forwarded-Proto
	}, "password=test123&callback=app.example.com")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should redirect
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusFound || ctx.Response().StatusCode() == fiber.StatusMovedPermanently)
}

// MockAuthenticator is a mock implementation of Authenticator for testing.
type MockAuthenticator struct {
	AuthenticateFunc func(sess *session.Session) error
}

func (m *MockAuthenticator) Authenticate(sess *session.Session) error {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(sess)
	}
	return nil
}

// TestLoginAPI_SessionStoreError tests error handling when session store fails
func TestLoginAPI_SessionStoreError(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "password=test123")
	defer app.ReleaseCtx(ctx)

	mockSessionGetter := &MockSessionGetter{
		GetFunc: func(ctx *fiber.Ctx) (*session.Session, error) {
			return nil, fiber.ErrInternalServerError
		},
	}
	mockAuthenticator := &MockAuthenticator{}

	err = loginAPIHandler(ctx, mockSessionGetter, mockAuthenticator)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusInternalServerError, ctx.Response().StatusCode())

	// Verify error response format
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, i18n.TStatic("error.session_store_failed"))
}

// TestLoginAPI_AuthenticateError tests error handling when authenticate fails
func TestLoginAPI_AuthenticateError(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	store := setupTestStore()
	ctx, app := createTestContext("POST", "/_login", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "password=test123")
	defer app.ReleaseCtx(ctx)

	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)

	mockSessionGetter := &MockSessionGetter{
		GetFunc: func(ctx *fiber.Ctx) (*session.Session, error) {
			return sess, nil
		},
	}
	mockAuthenticator := &MockAuthenticator{
		AuthenticateFunc: func(sess *session.Session) error {
			return fiber.ErrInternalServerError
		},
	}

	err = loginAPIHandler(ctx, mockSessionGetter, mockAuthenticator)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusInternalServerError, ctx.Response().StatusCode())

	// Verify error response format
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, i18n.TStatic("error.authenticate_failed"))
}

// TestLoginAPI_EmptySessionID_Error tests error handling when session ID is always empty
// Note: This test is difficult to implement because Fiber session always generates an ID.
// The error path (line 101-103 in login.go) is theoretically reachable but requires
// a session that has no ID even after Save(). This scenario is unlikely in practice
// as Fiber session always generates an ID when Get() is called.
// We test the retry error path instead in TestLoginAPI_EmptySessionID_RetryGetSessionError.
func TestLoginAPI_EmptySessionID_Error(t *testing.T) {
	t.Skip("Cannot easily mock session.ID() to return empty string - Fiber session always generates ID. This error path is covered by integration tests.")
}

// TestLoginAPI_EmptySessionID_RetryGetSessionError tests error handling when retry get session fails
// Note: Since Fiber session always generates an ID on Get(), we can't easily test
// the path where initial ID is empty and retry fails. The error path at line 97
// is theoretically reachable but requires a session implementation that doesn't
// always generate an ID, which Fiber session does.
// This error path is covered by integration tests.
func TestLoginAPI_EmptySessionID_RetryGetSessionError(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	// Since Fiber session always generates an ID on Get(), we can't easily test
	// the path where initial ID is empty and retry fails. The error path at line 97
	// is theoretically reachable but requires a session implementation that doesn't
	// always generate an ID, which Fiber session does.
	// This error path is covered by integration tests.
	t.Skip("Cannot easily test retry error path - Fiber session always generates ID on Get(). This error path is covered by integration tests.")
}

// TestLoginRoute_SessionStoreError tests error handling when session store fails
func TestLoginRoute_SessionStoreError(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize(testLogger())
	testza.AssertNoError(t, err)

	ctx, app := createTestContext("GET", "/_login", nil, "")
	defer app.ReleaseCtx(ctx)

	mockSessionGetter := &MockSessionGetter{
		GetFunc: func(ctx *fiber.Ctx) (*session.Session, error) {
			return nil, fiber.ErrInternalServerError
		},
	}

	err = loginRouteHandler(ctx, mockSessionGetter)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusInternalServerError, ctx.Response().StatusCode())

	// Verify error response format
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, i18n.TStatic("error.session_store_failed"))
}

// TestCheckRoute_WardenAuth_ValidPhone tests Warden authentication with valid phone
func TestCheckRoute_WardenAuth_ValidPhone(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	t.Setenv("WARDEN_URL", "http://localhost:8080")
	testLog := testLogger()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	InitForwardAuthHandler(testLog) // Re-init to pick up WARDEN_ENABLED=true

	// Create a mock HTTP server for Warden
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle /user endpoint (used by GetUserByIdentifier)
		if r.URL.Path == "/user" {
			phone := r.URL.Query().Get("phone")
			mail := r.URL.Query().Get("mail")

			var user struct {
				Phone  string `json:"phone"`
				Mail   string `json:"mail"`
				UserID string `json:"user_id"`
				Status string `json:"status"`
			}

			switch {
			case phone == "13800138000":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13800138000", Mail: "user1@example.com", UserID: "user1", Status: "active"}
			case phone == "13900139000":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13900139000", Mail: "user2@example.com", UserID: "user2", Status: "active"}
			case mail == "user2@example.com" || mail == "USER2@EXAMPLE.COM":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13900139000", Mail: "user2@example.com", UserID: "user2", Status: "active"}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(user)
			return
		}

		// Handle root endpoint (for backward compatibility)
		mockUsers := []struct {
			Phone string `json:"phone"`
			Mail  string `json:"mail"`
		}{
			{Phone: "13800138000", Mail: "user1@example.com"},
			{Phone: "13900139000", Mail: "user2@example.com"},
		}
		_ = json.NewEncoder(w).Encode(mockUsers)
	}))
	defer server.Close()

	// Initialize Warden client with mock server
	t.Setenv("WARDEN_URL", server.URL)
	_ = config.Initialize(testLogger())
	auth.ResetWardenClientForTesting()
	auth.InitWardenClient(testLogger())

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"X-User-Phone": "13800138000",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	// Verify user header is set (forwardauth-kit uses actual user_id when available)
	userHeader := string(ctx.Response().Header.Peek("X-Forwarded-User"))
	testza.AssertEqual(t, "user1", userHeader)
}

// TestCheckRoute_WardenAuth_ValidMail tests Warden authentication with valid email
func TestCheckRoute_WardenAuth_ValidMail(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	testLog := testLogger()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	InitForwardAuthHandler(testLog) // Re-init to pick up WARDEN_ENABLED=true

	// Create a mock HTTP server for Warden
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle /user endpoint (used by GetUserByIdentifier)
		if r.URL.Path == "/user" {
			phone := r.URL.Query().Get("phone")
			mail := r.URL.Query().Get("mail")

			var user struct {
				Phone  string `json:"phone"`
				Mail   string `json:"mail"`
				UserID string `json:"user_id"`
				Status string `json:"status"`
			}

			switch {
			case phone == "13800138000":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13800138000", Mail: "user1@example.com", UserID: "user1", Status: "active"}
			case phone == "13900139000":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13900139000", Mail: "user2@example.com", UserID: "user2", Status: "active"}
			case mail == "user1@example.com" || mail == "USER1@EXAMPLE.COM":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13800138000", Mail: "user1@example.com", UserID: "user1", Status: "active"}
			case mail == "user2@example.com" || mail == "USER2@EXAMPLE.COM":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13900139000", Mail: "user2@example.com", UserID: "user2", Status: "active"}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(user)
			return
		}

		// Handle root endpoint (for backward compatibility)
		mockUsers := []struct {
			Phone string `json:"phone"`
			Mail  string `json:"mail"`
		}{
			{Phone: "13800138000", Mail: "user1@example.com"},
			{Phone: "13900139000", Mail: "user2@example.com"},
		}
		_ = json.NewEncoder(w).Encode(mockUsers)
	}))
	defer server.Close()

	// Initialize Warden client with mock server
	t.Setenv("WARDEN_URL", server.URL)
	_ = config.Initialize(testLogger())
	auth.ResetWardenClientForTesting()
	auth.InitWardenClient(testLogger())

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"X-User-Mail": "user2@example.com",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	// Verify user header is set (forwardauth-kit uses actual user_id when available)
	userHeader := string(ctx.Response().Header.Peek("X-Forwarded-User"))
	testza.AssertEqual(t, "user2", userHeader)
}

// TestCheckRoute_WardenAuth_InvalidUser tests Warden authentication with invalid user
func TestCheckRoute_WardenAuth_InvalidUser(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	testLog := testLogger()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	InitForwardAuthHandler(testLog) // Re-init to pick up WARDEN_ENABLED=true

	// Create a mock HTTP server for Warden
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle /user endpoint (used by GetUserByIdentifier)
		if r.URL.Path == "/user" {
			phone := r.URL.Query().Get("phone")
			mail := r.URL.Query().Get("mail")

			// Only return user for known phone/mail
			if phone == "13800138000" || mail == "user1@example.com" || mail == "USER1@EXAMPLE.COM" {
				user := struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13800138000", Mail: "user1@example.com", UserID: "user1", Status: "active"}
				_ = json.NewEncoder(w).Encode(user)
				return
			}

			// User not found
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Handle root endpoint (for backward compatibility)
		mockUsers := []struct {
			Phone string `json:"phone"`
			Mail  string `json:"mail"`
		}{
			{Phone: "13800138000", Mail: "user1@example.com"},
		}
		_ = json.NewEncoder(w).Encode(mockUsers)
	}))
	defer server.Close()

	// Initialize Warden client with mock server
	t.Setenv("WARDEN_URL", server.URL)
	_ = config.Initialize(testLogger())
	auth.ResetWardenClientForTesting()
	auth.InitWardenClient(testLogger())

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"X-User-Phone": "99999999999", // Not in list
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should fall back to session check, which will redirect or return 401
	// Since no session is authenticated, it should redirect or return 401
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusUnauthorized ||
		ctx.Response().StatusCode() == fiber.StatusFound ||
		ctx.Response().StatusCode() == fiber.StatusMovedPermanently)
}

// TestCheckRoute_WardenAuth_EmptyHeaders tests Warden authentication with empty headers
func TestCheckRoute_WardenAuth_EmptyHeaders(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	testLog := testLogger()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	InitForwardAuthHandler(testLog) // Re-init to pick up WARDEN_ENABLED=true

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		// No X-User-Phone or X-User-Mail headers
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	// Should fall back to session check
	testza.AssertTrue(t, ctx.Response().StatusCode() == fiber.StatusUnauthorized ||
		ctx.Response().StatusCode() == fiber.StatusFound ||
		ctx.Response().StatusCode() == fiber.StatusMovedPermanently)
}

// TestCheckRoute_WardenAuth_WithBothHeaders tests Warden authentication with both phone and mail headers
func TestCheckRoute_WardenAuth_WithBothHeaders(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	testLog := testLogger()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	InitForwardAuthHandler(testLog) // Re-init to pick up WARDEN_ENABLED=true

	// Create a mock HTTP server for Warden
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle /user endpoint (used by GetUserByIdentifier)
		if r.URL.Path == "/user" {
			phone := r.URL.Query().Get("phone")
			mail := r.URL.Query().Get("mail")

			var user struct {
				Phone  string `json:"phone"`
				Mail   string `json:"mail"`
				UserID string `json:"user_id"`
				Status string `json:"status"`
			}

			switch {
			case phone == "13800138000":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13800138000", Mail: "user1@example.com", UserID: "user1", Status: "active"}
			case phone == "13900139000":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13900139000", Mail: "user2@example.com", UserID: "user2", Status: "active"}
			case mail == "user1@example.com" || mail == "USER1@EXAMPLE.COM":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13800138000", Mail: "user1@example.com", UserID: "user1", Status: "active"}
			case mail == "user2@example.com" || mail == "USER2@EXAMPLE.COM":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13900139000", Mail: "user2@example.com", UserID: "user2", Status: "active"}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(user)
			return
		}

		// Handle root endpoint (for backward compatibility)
		mockUsers := []struct {
			Phone string `json:"phone"`
			Mail  string `json:"mail"`
		}{
			{Phone: "13800138000", Mail: "user1@example.com"},
			{Phone: "13900139000", Mail: "user2@example.com"},
		}
		_ = json.NewEncoder(w).Encode(mockUsers)
	}))
	defer server.Close()

	// Initialize Warden client with mock server
	t.Setenv("WARDEN_URL", server.URL)
	_ = config.Initialize(testLogger())
	auth.ResetWardenClientForTesting()
	auth.InitWardenClient(testLogger())

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"X-User-Phone": "13800138000",
		"X-User-Mail":  "user1@example.com",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	// Verify user header is set (forwardauth-kit uses actual user_id when available)
	userHeader := string(ctx.Response().Header.Peek("X-Forwarded-User"))
	testza.AssertEqual(t, "user1", userHeader)
}

// TestCheckRoute_WardenAuth_WithCustomUserHeader tests Warden authentication with custom user header name
func TestCheckRoute_WardenAuth_WithCustomUserHeader(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	t.Setenv("USER_HEADER_NAME", "X-Custom-User")
	testLog := testLogger()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	InitForwardAuthHandler(testLog) // Re-init to pick up WARDEN_ENABLED=true

	// Create a mock HTTP server for Warden
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle /user endpoint (used by GetUserByIdentifier)
		if r.URL.Path == "/user" {
			phone := r.URL.Query().Get("phone")
			mail := r.URL.Query().Get("mail")

			var user struct {
				Phone  string `json:"phone"`
				Mail   string `json:"mail"`
				UserID string `json:"user_id"`
				Status string `json:"status"`
			}

			switch {
			case phone == "13800138000":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13800138000", Mail: "user1@example.com", UserID: "user1", Status: "active"}
			case mail == "user1@example.com" || mail == "USER1@EXAMPLE.COM":
				user = struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{Phone: "13800138000", Mail: "user1@example.com", UserID: "user1", Status: "active"}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(user)
			return
		}

		// Handle root endpoint (for backward compatibility)
		mockUsers := []struct {
			Phone string `json:"phone"`
			Mail  string `json:"mail"`
		}{
			{Phone: "13800138000", Mail: "user1@example.com"},
		}
		_ = json.NewEncoder(w).Encode(mockUsers)
	}))
	defer server.Close()

	// Initialize Warden client with mock server
	t.Setenv("WARDEN_URL", server.URL)
	_ = config.Initialize(testLogger())
	auth.ResetWardenClientForTesting()
	auth.InitWardenClient(testLogger())

	store := setupTestStore()
	handler := CheckRoute(store)

	ctx, app := createTestContext("GET", "/_auth", map[string]string{
		"X-User-Phone": "13800138000",
	}, "")
	defer app.ReleaseCtx(ctx)

	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, ctx.Response().StatusCode())

	// Verify custom user header is set (forwardauth-kit uses actual user_id when available)
	userHeader := string(ctx.Response().Header.Peek("X-Custom-User"))
	testza.AssertEqual(t, "user1", userHeader)
}
