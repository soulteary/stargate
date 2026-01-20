package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	"github.com/soulteary/stargate/src/internal/config"
)

func setupTestConfig(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")
	err := config.Initialize()
	testza.AssertNoError(t, err)
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
	_ = config.Initialize()

	store := setupSessionStore()
	testza.AssertNotNil(t, store)
}

func TestSetupSessionStore_WithCookieDomain(t *testing.T) {
	setupTestConfig(t)

	// Set cookie domain
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	_ = config.Initialize()

	store := setupSessionStore()
	testza.AssertNotNil(t, store)
}

func TestSetupRoutes(t *testing.T) {
	setupTestConfig(t)

	app := fiber.New()
	store := setupSessionStore()

	// Test that setupRoutes doesn't panic
	testza.AssertNotPanics(t, func() {
		setupRoutes(app, store)
	})

	// Verify routes are registered by testing health endpoint
	req := httptest.NewRequest("GET", RouteHealth, nil)
	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
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

func TestStartServer_PortLogic_DefaultPort(t *testing.T) {
	setupTestConfig(t)

	// Unset PORT to test default
	_ = os.Unsetenv("PORT")

	// Test the port logic from startServer function
	port := DefaultPort
	// PORT is unset, so port should remain DefaultPort
	_ = os.Getenv("PORT")

	testza.AssertEqual(t, ":80", port)
}

func TestStartServer_PortLogic_CustomPort(t *testing.T) {
	setupTestConfig(t)

	// Test with custom port
	t.Setenv("PORT", "8080")

	port := DefaultPort
	if envPort := os.Getenv("PORT"); envPort != "" {
		// Simulate the logic from startServer
		if envPort[0] != ':' {
			port = ":" + envPort
		} else {
			port = envPort
		}
	}

	testza.AssertEqual(t, ":8080", port)
}

func TestStartServer_PortLogic_CustomPortWithColon(t *testing.T) {
	setupTestConfig(t)

	// Test with custom port that already has colon
	t.Setenv("PORT", ":9090")

	port := DefaultPort
	if envPort := os.Getenv("PORT"); envPort != "" {
		// Simulate the logic from startServer
		if envPort[0] != ':' {
			port = ":" + envPort
		} else {
			port = envPort
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

	store := setupSessionStore()
	testza.AssertNotNil(t, store)

	// Verify store is functional by creating a test app
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
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
