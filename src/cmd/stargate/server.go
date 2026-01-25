package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/gofiber/template/html"
	"github.com/sirupsen/logrus"
	"github.com/soulteary/cli-kit/env"
	i18nkit "github.com/soulteary/i18n-kit"
	metricskit "github.com/soulteary/metrics-kit"
	middlewarekit "github.com/soulteary/middleware-kit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/handlers"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/metrics"
	"github.com/soulteary/stargate/src/internal/storage"
	internal_tracing "github.com/soulteary/stargate/src/internal/tracing"
)

// findTemplatesPath finds the correct path to templates directory.
// It checks both ./internal/web/templates (for local development) and ./web/templates (for Docker).
func findTemplatesPath() string {
	paths := []string{
		"./internal/web/templates",
		"./web/templates",
		"./src/internal/web/templates",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			logrus.Debug("Found templates at: ", absPath)
			return path
		}
	}
	// Default to internal path for local development
	return "./internal/web/templates"
}

// setupTemplates initializes the HTML template engine.
// It loads templates from the web/templates directory.
func setupTemplates() *html.Engine {
	logrus.Debug("initializing html templating")
	templatesPath := findTemplatesPath()
	return html.New(templatesPath, ".html")
}

// setupSessionStore initializes the session store with configured settings.
// It sets up cookie-based session management with configurable domain support.
// If Redis storage is enabled via SESSION_STORAGE_ENABLED=true, it will use Redis for session storage.
func setupSessionStore() *session.Store {
	logrus.Debug("initializing session store")

	sessionConfig := session.Config{
		Expiration:     config.SessionExpiration,
		KeyLookup:      "cookie:" + auth.SessionCookieName,
		CookiePath:     "/",
		KeyGenerator:   utils.UUID,
		CookieHTTPOnly: true,
		CookieSameSite: fiber.CookieSameSiteLaxMode,
	}

	// If Cookie domain is configured, set it
	if config.CookieDomain.Value != "" {
		sessionConfig.CookieDomain = config.CookieDomain.Value
	}

	// Check if Redis session storage is enabled
	if config.SessionStorageEnabled.ToBool() {
		logrus.Info("Redis session storage is enabled, initializing Redis client...")

		// Parse Redis DB number
		redisDB := 0
		if config.SessionStorageRedisDB.Value != "" {
			if db, err := strconv.Atoi(config.SessionStorageRedisDB.Value); err == nil {
				redisDB = db
			} else {
				logrus.Warnf("Invalid SESSION_STORAGE_REDIS_DB value '%s', using default 0", config.SessionStorageRedisDB.Value)
			}
		}

		// Create Redis client
		redisClient, err := storage.NewRedisClientFromConfig(
			config.SessionStorageRedisAddr.Value,
			config.SessionStorageRedisPassword.Value,
			redisDB,
		)
		if err != nil {
			logrus.Fatalf("Failed to initialize Redis client for session storage: %v", err)
		}

		// Create Redis storage
		redisStorage := storage.NewRedisStorage(
			redisClient,
			config.SessionStorageRedisKeyPrefix.Value,
		)

		// Set the storage in session config
		sessionConfig.Storage = redisStorage
		logrus.Info("Session storage configured to use Redis")
	} else {
		logrus.Debug("Using default in-memory session storage")
	}

	return session.New(sessionConfig)
}

// setupRoutes registers all HTTP routes for the application.
// This includes authentication, login, logout, session exchange, and health check endpoints.
func setupRoutes(app *fiber.App, store *session.Store) {
	logrus.Debug("registering routes")
	// Initialize Herald client
	handlers.InitHeraldClient()

	app.Get(RouteHealth, handlers.HealthRoute())
	app.Get(RouteRoot, handlers.IndexRoute(store))
	app.Get(RouteLogin, handlers.LoginRoute(store))
	app.Post(RouteLogin, handlers.LoginAPI(store))
	app.Post("/_send_verify_code", handlers.SendVerifyCodeAPI())
	app.Get(RouteLogout, handlers.LogoutRoute(store))
	app.Get(RouteSessionExchange, handlers.SessionShareRoute())
	app.Get(RouteAuth, handlers.CheckRoute(store))
	// Prometheus metrics endpoint
	app.Get("/metrics", metricskit.FiberHandlerFor(metrics.Registry))
}

// findAssetsPath finds the correct path to assets directory.
func findAssetsPath() string {
	paths := []string{
		"./internal/web/templates/assets",
		"./web/templates/assets",
		"./src/internal/web/templates/assets",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	// Default to internal path for local development
	return "./internal/web/templates/assets"
}

// findFaviconPath finds the correct path to favicon file.
func findFaviconPath() string {
	paths := []string{
		"./internal/web/templates/assets/favicon.ico",
		"./web/templates/assets/favicon.ico",
		"./src/internal/web/templates/assets/favicon.ico",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	// Default to internal path for local development
	return "./internal/web/templates/assets/favicon.ico"
}

// setupStaticFiles registers static file serving for assets.
func setupStaticFiles(app *fiber.App) {
	logrus.Debug("registering static file server for assets")
	assetsPath := findAssetsPath()
	app.Static("/assets", assetsPath)
}

// setupMiddleware configures all middleware for the Fiber application.
// This includes recover, security headers, logging, tracing, i18n, and favicon handling.
func setupMiddleware(app *fiber.App) {
	// 1. Panic recovery (highest priority - prevents server crashes)
	app.Use(recover.New())
	logrus.Debug("Panic recovery middleware enabled")

	// 2. Security headers (XSS protection, clickjacking prevention, etc.)
	app.Use(middlewarekit.SecurityHeaders(middlewarekit.DefaultSecurityHeadersConfig()))
	logrus.Debug("Security headers middleware enabled")

	// 3. OpenTelemetry tracing middleware (if enabled)
	if config.OTLPEnabled.ToBool() {
		app.Use(internal_tracing.TracingMiddleware("stargate"))
		logrus.Info("OpenTelemetry tracing middleware enabled")
	}

	// 4. i18n middleware (language detection from Query > Cookie > Header > Accept-Language)
	app.Use(i18nkit.FiberMiddleware(i18nkit.MiddlewareConfig{
		Bundle: i18n.GetBundle(),
	}))
	logrus.Debug("i18n middleware enabled")

	// 5. Request logging with middleware-kit (structured logging with zerolog)
	app.Use(middlewarekit.RequestLogging(middlewarekit.LoggingConfig{
		Logger:         &zerologLogger,
		SkipPaths:      []string{"/healthz", "/metrics"},
		IncludeLatency: true,
	}))
	logrus.Debug("Request logging middleware enabled")

	// 6. Rate limiting (optional - uncomment to enable for production)
	// To enable rate limiting, uncomment the following code:
	//
	// limiter := middlewarekit.NewRateLimiter(middlewarekit.RateLimiterConfig{
	// 	Rate:            100,           // 100 requests per minute
	// 	Window:          time.Minute,
	// 	MaxVisitors:     10000,
	// 	CleanupInterval: time.Minute,
	// })
	// app.Use(middlewarekit.RateLimit(middlewarekit.RateLimitConfig{
	// 	Limiter:   limiter,
	// 	SkipPaths: []string{"/healthz", "/metrics"},
	// 	Logger:    &zerologLogger,
	// 	OnLimitReached: func(key string) {
	// 		// Optional: increment Prometheus counter
	// 		// metrics.RateLimitExceeded.Inc()
	// 	},
	// }))
	// logrus.Info("Rate limiting middleware enabled")

	// 7. Favicon middleware
	logrus.Debug("adding favicon middleware")
	faviconPath := findFaviconPath()
	// Only add favicon middleware if the file exists
	if _, err := os.Stat(faviconPath); err == nil {
		app.Use(favicon.New(favicon.Config{
			File: faviconPath,
		}))
	} else {
		logrus.Debug("Favicon file not found, skipping favicon middleware: ", faviconPath)
	}
}

// createApp creates and configures a new Fiber application.
// It sets up templates, middleware, routes, and static file serving.
//
// Returns a fully configured Fiber app ready to start.
func createApp() *fiber.App {
	engine := setupTemplates()

	logrus.Debug("creating web server instance")
	app := fiber.New(fiber.Config{
		Views:                 engine,
		DisableStartupMessage: true,
	})

	setupMiddleware(app)
	store := setupSessionStore()
	setupRoutes(app, store)
	setupStaticFiles(app)

	return app
}

// startServer starts the HTTP server on the default port.
//
// Parameters:
//   - app: The configured Fiber application
//
// Returns an error if the server cannot be started.
func startServer(app *fiber.App) error {
	port := DefaultPort
	// Support overriding default port via PORT environment variable (for local testing)
	// Use cli-kit env.GetTrimmed for consistent handling
	if envPort := env.GetTrimmed("PORT", ""); envPort != "" {
		if !strings.HasPrefix(envPort, ":") {
			port = ":" + envPort
		} else {
			port = envPort
		}
		logrus.Info("Using custom port from PORT environment variable: ", port)
	}
	logrus.Debug("starting web server on port: ", port)
	return app.Listen(port)
}
