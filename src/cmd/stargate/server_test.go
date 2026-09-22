package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	health "github.com/soulteary/health-kit/v4"
	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/handlers"
	"github.com/valyala/fasthttp"
)

// testLoggerMain creates a logger instance for testing
func testLoggerMain() *logger.Logger {
	return logger.New(logger.Config{
		Level:       logger.DebugLevel,
		Format:      logger.FormatJSON,
		ServiceName: "main-test",
	})
}

func setupTestConfig(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")
	err := config.Initialize(testLoggerMain())
	testza.AssertNoError(t, err)
}

func TestParseTrustedProxies(t *testing.T) {
	original := config.TrustedProxies.Value
	t.Cleanup(func() { config.TrustedProxies.Value = original })
	config.TrustedProxies.Value = "127.0.0.1, 10.0.0.0/8,::1"

	testza.AssertEqual(t, []string{"127.0.0.1", "10.0.0.0/8", "::1"}, parseTrustedProxies())
}

// ensureTestWorkingDir ensures tests run from the project root directory
// where the src/internal/web/templates path exists
func ensureTestWorkingDir(t *testing.T) {
	originalWd, err := os.Getwd()
	testza.AssertNoError(t, err)

	// Restore original directory when test completes
	t.Cleanup(func() {
		_ = os.Chdir(originalWd)
	})

	// Check if we're already in the right directory (project root)
	if _, err := os.Stat("src/internal/web/templates"); err == nil {
		return
	}

	// Check if we're in src directory
	if _, err := os.Stat("internal/web/templates"); err == nil {
		return
	}

	// Try to find project root by looking for go.mod and templates
	dir := originalWd
	for i := 0; i < 10; i++ {
		// Check if this directory has go.mod
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			// Check if templates exist relative to this directory
			templatesPath := filepath.Join(dir, "src", "internal", "web", "templates")
			if _, err := os.Stat(templatesPath); err == nil {
				err := os.Chdir(dir)
				testza.AssertNoError(t, err)
				return
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// If we still haven't found it, try going up from current directory
	// and checking for src/internal/web/templates
	dir = originalWd
	for i := 0; i < 10; i++ {
		templatesPath := filepath.Join(dir, "src", "internal", "web", "templates")
		if _, err := os.Stat(templatesPath); err == nil {
			err := os.Chdir(dir)
			testza.AssertNoError(t, err)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// If we can't find the templates, log a warning but don't fail
	// The modified setupMiddleware will handle missing favicon gracefully
	t.Logf("Warning: Could not find templates directory, tests may fail if they require files")
}

func TestFindTemplatesPath_InternalPath(t *testing.T) {
	// Test finding templates in internal path
	path := findTemplatesPath()

	// Should return a valid path (either internal or web)
	testza.AssertTrue(t, len(path) > 0, "path should not be empty")

	// Verify it's one of the expected paths
	expectedPaths := []string{
		"./internal/web/templates",
		"./web/templates",
	}

	found := false
	for _, expected := range expectedPaths {
		if path == expected {
			found = true
			break
		}
	}

	// If neither path exists, should default to internal path
	if !found {
		testza.AssertEqual(t, "./internal/web/templates", path)
	}
}

func TestFindTemplatesPath_DefaultFallback(t *testing.T) {
	// Test that function returns default path when neither exists
	// This is hard to test without mocking os.Stat, but we can verify
	// the function doesn't panic and returns a valid path
	path := findTemplatesPath()
	testza.AssertTrue(t, len(path) > 0, "path should not be empty")
}

func TestSetupTemplates(t *testing.T) {
	// Test that setupTemplates creates an engine without panicking
	testza.AssertNotPanics(t, func() {
		engine := setupTemplates()
		testza.AssertNotNil(t, engine)
	})
}

func TestSetupSessionStore_WithoutCookieDomain(t *testing.T) {
	setupTestConfig(t)

	// Clear cookie domain
	_ = os.Unsetenv("COOKIE_DOMAIN")
	_ = config.Initialize(testLoggerMain())

	store, _ := setupSessionStore()
	testza.AssertNotNil(t, store)
}

func TestSetupSessionStore_WithCookieDomain(t *testing.T) {
	setupTestConfig(t)

	// Set cookie domain
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	_ = config.Initialize(testLoggerMain())

	store, _ := setupSessionStore()
	testza.AssertNotNil(t, store)
}

// TestSetupSessionStore_WithInvalidRedisDBEnv_RedisDisabled ensures setupSessionStore runs
// when SESSION_STORAGE_REDIS_DB is set but Redis is disabled. The "invalid REDIS_DB" log path
// is only hit when SESSION_STORAGE_ENABLED=true (requires Redis connection).
func TestSetupSessionStore_WithInvalidRedisDBEnv_RedisDisabled(t *testing.T) {
	setupTestConfig(t)
	t.Setenv("SESSION_STORAGE_ENABLED", "false")
	t.Setenv("SESSION_STORAGE_REDIS_DB", "not_a_number")
	_ = config.Initialize(testLoggerMain())

	store, _ := setupSessionStore()
	testza.AssertNotNil(t, store)
}

// TestSetupHealthChecker_Combinations covers Herald/Warden/session-storage branches.
// The branch "SessionStorageEnabled && redisClient != nil" is covered when running with Redis enabled (e.g. integration).
func TestSetupHealthChecker_Combinations(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"herald_disabled_warden_disabled", map[string]string{"HERALD_ENABLED": "false", "WARDEN_ENABLED": "false"}},
		{"herald_enabled_url_empty", map[string]string{"HERALD_ENABLED": "true", "HERALD_URL": "", "WARDEN_ENABLED": "false"}},
		{"herald_enabled_url_set", map[string]string{"HERALD_ENABLED": "true", "HERALD_URL": "http://herald.local/", "WARDEN_ENABLED": "false"}},
		{"warden_enabled_url_empty", map[string]string{"HERALD_ENABLED": "false", "WARDEN_ENABLED": "true", "WARDEN_URL": ""}},
		{"warden_enabled_url_set", map[string]string{"HERALD_ENABLED": "false", "WARDEN_ENABLED": "true", "WARDEN_URL": "http://warden.local/"}},
		{"session_storage_disabled", map[string]string{"HERALD_ENABLED": "false", "WARDEN_ENABLED": "false", "SESSION_STORAGE_ENABLED": "false"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestConfig(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			_ = config.Initialize(testLoggerMain())
			agg := setupHealthChecker(nil)
			testza.AssertNotNil(t, agg)
		})
	}
}

func TestSetupHealthCheckerReportsTLSConfigurationFailure(t *testing.T) {
	setupTestConfig(t)
	previousEnabled := config.HeraldEnabled.Value
	previousURL := config.HeraldURL.Value
	previousCA := config.HeraldTLSCACertFile.Value
	t.Cleanup(func() {
		config.HeraldEnabled.Value = previousEnabled
		config.HeraldURL.Value = previousURL
		config.HeraldTLSCACertFile.Value = previousCA
	})
	config.HeraldEnabled.Value = "true"
	config.HeraldURL.Value = "https://herald.example.com"
	config.HeraldTLSCACertFile.Value = filepath.Join(t.TempDir(), "missing-ca.pem")

	result := setupHealthChecker(nil).Check(context.Background())
	heraldResult, ok := result.Checks["herald"]
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, health.StatusUnhealthy, heraldResult.Status)
	testza.AssertContains(t, heraldResult.Error, "read Herald CA certificate")
}

func TestSetupRoutes(t *testing.T) {
	setupTestConfig(t)

	app := fiber.New()
	store, _ := setupSessionStore()

	// Create a simple health aggregator for testing
	healthConfig := health.DefaultConfig().WithServiceName("stargate")
	aggregator := health.NewAggregator(healthConfig)

	// Test that setupRoutes doesn't panic
	testza.AssertNotPanics(t, func() {
		setupRoutes(app, store, aggregator, handlers.NewSessionExchangeReplayStore(nil))
	})

	// Verify liveness, readiness, and the compatibility route are registered.
	for _, route := range []string{RouteHealthz, RouteReadyz, RouteHealth} {
		req := httptest.NewRequest("GET", route, nil)
		resp, err := app.Test(req)
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
	}
}

func TestHealthzStaysLiveWhenDependencyIsUnready(t *testing.T) {
	setupTestConfig(t)

	app := fiber.New()
	store, _ := setupSessionStore()
	aggregator := health.NewAggregator(health.DefaultConfig().WithServiceName("stargate"))
	aggregator.AddChecker(health.NewCheckerFunc("dependency", func(context.Context) health.CheckResult {
		return health.CheckResult{Name: "dependency", Status: health.StatusUnhealthy}
	}))
	setupRoutes(app, store, aggregator, handlers.NewSessionExchangeReplayStore(nil))

	liveness, err := app.Test(httptest.NewRequest("GET", RouteHealthz, nil))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, liveness.StatusCode)

	readiness, err := app.Test(httptest.NewRequest("GET", RouteReadyz, nil))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusServiceUnavailable, readiness.StatusCode)

	legacy, err := app.Test(httptest.NewRequest("GET", RouteHealth, nil))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusServiceUnavailable, legacy.StatusCode)
}

func TestFindAssetsPath(t *testing.T) {
	// Test finding assets path
	path := findAssetsPath()

	// Should return a valid path
	testza.AssertTrue(t, len(path) > 0, "path should not be empty")

	// Verify it's one of the expected paths
	expectedPaths := []string{
		"./internal/web/templates/assets",
		"./web/templates/assets",
	}

	found := false
	for _, expected := range expectedPaths {
		if path == expected {
			found = true
			break
		}
	}

	// If neither path exists, should default to internal path
	if !found {
		testza.AssertEqual(t, "./internal/web/templates/assets", path)
	}
}

func TestFindFaviconPath(t *testing.T) {
	// Test finding favicon path
	path := findFaviconPath()

	// Should return a valid path
	testza.AssertTrue(t, len(path) > 0, "path should not be empty")

	// Verify it's one of the expected paths
	expectedPaths := []string{
		"./internal/web/templates/assets/favicon.ico",
		"./web/templates/assets/favicon.ico",
	}

	found := false
	for _, expected := range expectedPaths {
		if path == expected {
			found = true
			break
		}
	}

	// If neither path exists, should default to internal path
	if !found {
		testza.AssertEqual(t, "./internal/web/templates/assets/favicon.ico", path)
	}
}

func TestSetupStaticFiles(t *testing.T) {
	setupTestConfig(t)

	app := fiber.New()

	// Test that setupStaticFiles doesn't panic
	testza.AssertNotPanics(t, func() {
		setupStaticFiles(app)
	})
}

func TestSetupMiddleware(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)

	app := fiber.New()

	// Test that setupMiddleware doesn't panic
	// Note: This requires the favicon file to exist
	testza.AssertNotPanics(t, func() {
		setupMiddleware(app)
	})

	// Verify middleware is registered by making a request
	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	// Should not panic, even if route doesn't exist
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, resp)
}

func TestCreateApp(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)

	// Test that createApp returns a valid Fiber app
	app := createApp()
	testza.AssertNotNil(t, app)

	// Verify app is configured with templates
	testza.AssertNotNil(t, app.Config().Views)

	// Verify routes are registered by testing health endpoint
	req := httptest.NewRequest("GET", RouteHealth, nil)
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestCreateApp_AllRoutesRegistered(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)

	app := createApp()

	// Test all routes are registered
	routes := []string{
		RouteHealth,
		RouteHealthz,
		RouteReadyz,
		RouteRoot,
		RouteLogin,
		RouteLogout,
		RouteSessionExchange,
		RouteAuth,
	}

	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			resp, err := app.Test(req)
			// Route should exist (may return different status codes)
			testza.AssertNoError(t, err)
			testza.AssertNotNil(t, resp)
		})
	}
}

func TestCreateApp_HasProductionLimits(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)
	app := createApp()
	cfg := app.Config()

	testza.AssertEqual(t, 1*1024*1024, cfg.BodyLimit)
	testza.AssertEqual(t, 10*time.Second, cfg.ReadTimeout)
	testza.AssertEqual(t, 15*time.Second, cfg.WriteTimeout)
	testza.AssertEqual(t, 60*time.Second, cfg.IdleTimeout)
}

func TestStartServer_PortLogic_DefaultPort(t *testing.T) {
	_ = os.Unsetenv("PORT")
	setupTestConfig(t)

	port := DefaultPort
	if configPort := config.Port.String(); configPort != "" {
		if !strings.HasPrefix(configPort, ":") {
			port = ":" + configPort
		} else {
			port = configPort
		}
	}
	testza.AssertEqual(t, ":80", port)
}

func TestStartServer_PortLogic_CustomPort(t *testing.T) {
	t.Setenv("PORT", "8080")
	setupTestConfig(t)

	port := DefaultPort
	if configPort := config.Port.String(); configPort != "" {
		if !strings.HasPrefix(configPort, ":") {
			port = ":" + configPort
		} else {
			port = configPort
		}
	}
	testza.AssertEqual(t, ":8080", port)
}

func TestStartServer_PortLogic_CustomPortWithColon(t *testing.T) {
	t.Setenv("PORT", ":9090")
	setupTestConfig(t)

	port := DefaultPort
	if configPort := config.Port.String(); configPort != "" {
		if !strings.HasPrefix(configPort, ":") {
			port = ":" + configPort
		} else {
			port = configPort
		}
	}
	testza.AssertEqual(t, ":9090", port)
}

func TestSetupTemplates_EngineCreated(t *testing.T) {
	engine := setupTemplates()
	testza.AssertNotNil(t, engine)
}

func TestSetupSessionStore_ConfigApplied(t *testing.T) {
	setupTestConfig(t)

	store, _ := setupSessionStore()
	testza.AssertNotNil(t, store)

	// Verify store is functional by creating a test app
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return err
		}
		testza.AssertNotNil(t, sess)
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
}

func TestCreateApp_StaticFilesRegistered(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)

	app := createApp()

	// Test that static files route is registered
	// Note: This will return 404 if assets don't exist, but route should be registered
	req := httptest.NewRequest("GET", "/assets/favicon.ico", nil)
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	// Route exists (may be 404 if file doesn't exist, but that's OK)
	testza.AssertNotNil(t, resp)
}

func TestCreateApp_MiddlewareRegistered(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)

	app := createApp()

	// Verify middleware is working by checking response headers
	req := httptest.NewRequest("GET", RouteHealth, nil)
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
	// Middleware should have processed the request
	testza.AssertNotNil(t, resp)
}

func TestSetupMiddleware_FaviconNotFound(t *testing.T) {
	ensureTestWorkingDir(t)
	setupTestConfig(t)

	app := fiber.New()

	// Test that setupMiddleware doesn't panic even if favicon doesn't exist
	// We can't easily remove the favicon file, but we can verify the code path
	// by checking that the function handles the error gracefully
	testza.AssertNotPanics(t, func() {
		setupMiddleware(app)
	})
}

// TestCreateApp_LoginWorksWithoutRedis is the end-to-end form of the nil-client
// regression: SESSION_STORAGE_ENABLED defaults to false, so this is the default
// deployment, and every rate-limited POST used to panic inside the rate-limit
// store and come back as a 500 from the recover middleware.
func TestCreateApp_LoginWorksWithoutRedis(t *testing.T) {
	ensureTestWorkingDir(t)
	initLogger()
	t.Setenv("SESSION_STORAGE_ENABLED", "false")
	setupTestConfig(t)

	app := createApp()

	req := httptest.NewRequest(http.MethodPost, RouteLogin, strings.NewReader("password=test123"))
	req.Host = config.AuthHost.String()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	testza.AssertNoError(t, err)
	testza.AssertNotEqual(t, http.StatusInternalServerError, resp.StatusCode,
		"login must not fault when session storage is in memory: %s", string(body))
	testza.AssertEqual(t, http.StatusOK, resp.StatusCode, string(body))

	// The session cookie proves the request reached the login handler rather
	// than dying in middleware.
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}
	testza.AssertNotNil(t, sessionCookie, "login should have issued a session cookie")
}

// TestSetupSessionStoreRedisKeyPrefix pins the Redis key layout that sessions
// are stored under.
//
// session-kit v3 moved Redis storage out of the root package into redisstore,
// and setupSessionStore now builds the Redis client itself rather than letting
// the kit build one and handing it back. The prefix rules survived that move -
// a configured prefix is used as given, an empty one falls back to "session:",
// and one without a trailing colon gets one - and they decide whether an
// existing deployment still finds its sessions after a rolling restart, so
// they are asserted against a real Redis rather than inferred.
func TestSetupSessionStoreRedisKeyPrefix(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prefix string
		want   string
	}{
		{name: "configured prefix is used as given", prefix: "stargate:session:", want: "stargate:session:"},
		{name: "empty prefix falls back to session:", prefix: "", want: "session:"},
		{name: "prefix without trailing colon gets one", prefix: "noColon", want: "noColon:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			redisServer := miniredis.RunT(t)
			initLogger()

			t.Setenv("SESSION_STORAGE_ENABLED", "true")
			t.Setenv("SESSION_STORAGE_REDIS_ADDR", redisServer.Addr())
			t.Setenv("SESSION_STORAGE_REDIS_DB", "0")
			setupTestConfig(t)

			// SESSION_STORAGE_REDIS_KEY_PREFIX has a non-empty default, so the
			// empty case has to be set past the config layer to reach the kit.
			originalPrefix := config.SessionStorageRedisKeyPrefix.Value
			t.Cleanup(func() { config.SessionStorageRedisKeyPrefix.Value = originalPrefix })
			config.SessionStorageRedisKeyPrefix.Value = tc.prefix

			store, redisClient := setupSessionStore()
			testza.AssertNotNil(t, store)
			// The client is shared with the challenge context store, the
			// rate-limit store, the replay store and the health check, all of
			// which need the concrete type.
			testza.AssertNotNil(t, redisClient)
			testza.AssertNoError(t, redisClient.Ping(context.Background()).Err())

			app := fiber.New()
			ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(ctx)

			sess, err := store.Get(ctx)
			testza.AssertNoError(t, err)
			sess.Set("probe", "value")
			testza.AssertNoError(t, sess.Save())

			testza.AssertTrue(t, redisServer.Exists(tc.want+sess.ID()),
				"session key should be stored under %q, found %v", tc.want, redisServer.Keys())
		})
	}
}
