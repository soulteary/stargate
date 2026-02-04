package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fibersession "github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/gofiber/template/html"
	"github.com/redis/go-redis/v9"
	health "github.com/soulteary/health-kit"
	i18nkit "github.com/soulteary/i18n-kit"
	logger "github.com/soulteary/logger-kit"
	metricskit "github.com/soulteary/metrics-kit"
	middlewarekit "github.com/soulteary/middleware-kit"
	session "github.com/soulteary/session-kit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/handlers"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/metrics"
	internal_tracing "github.com/soulteary/stargate/src/internal/tracing"
)

// findFirstExistingPath returns the first path in candidates that exists, or defaultPath if none exist.
func findFirstExistingPath(candidates []string, defaultPath string) string {
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return defaultPath
}

// findTemplatesPath finds the correct path to templates directory.
// It checks both ./internal/web/templates (for local development) and ./web/templates (for Docker).
func findTemplatesPath() string {
	path := findFirstExistingPath([]string{
		"./internal/web/templates",
		"./web/templates",
		"./src/internal/web/templates",
	}, "./internal/web/templates")
	if absPath, err := filepath.Abs(path); err == nil {
		log.Debug().Str("path", absPath).Msg("Found templates")
	}
	return path
}

// setupTemplates initializes the HTML template engine.
// It loads templates from the web/templates directory.
func setupTemplates() *html.Engine {
	log.Debug().Msg("Initializing html templating")
	templatesPath := findTemplatesPath()
	return html.New(templatesPath, ".html")
}

// setupSessionStore initializes the session store with configured settings.
// It sets up cookie-based session management with configurable domain support.
// If Redis storage is enabled via SESSION_STORAGE_ENABLED=true, it will use Redis for session storage.
// Returns the session store and the Redis client (non-nil only when Redis is enabled) for reuse by health check, avoiding a second connection.
func setupSessionStore() (*fibersession.Store, *redis.Client) {
	log.Debug().Msg("Initializing session store")

	// Create session-kit config with Stargate settings
	sessionConfig := session.DefaultConfig().
		WithExpiration(config.SessionExpiration).
		WithCookieName(auth.SessionCookieName).
		WithCookiePath("/").
		WithHTTPOnly(true).
		WithSameSite("Lax")

	// If Cookie domain is configured, set it
	if config.CookieDomain.Value != "" {
		sessionConfig = sessionConfig.WithCookieDomain(config.CookieDomain.Value)
	}

	// Create session-kit Manager with appropriate storage
	var sessionStorage session.Storage
	var err error
	var redisClient *redis.Client

	// Check if Redis session storage is enabled
	if config.SessionStorageEnabled.ToBool() {
		log.Info().Msg("Redis session storage is enabled, initializing Redis client...")

		// Parse Redis DB number
		redisDB := 0
		if config.SessionStorageRedisDB.Value != "" {
			if db, parseErr := strconv.Atoi(config.SessionStorageRedisDB.Value); parseErr == nil {
				redisDB = db
			} else {
				log.Warn().Str("value", config.SessionStorageRedisDB.Value).Msg("Invalid SESSION_STORAGE_REDIS_DB value, using default 0")
			}
		}

		// Use NewRedisStorageFromConfig once; reuse the same client for health check to avoid double connection
		redisStorage, err := session.NewRedisStorageFromConfig(
			config.SessionStorageRedisAddr.Value,
			config.SessionStorageRedisPassword.Value,
			redisDB,
			config.SessionStorageRedisKeyPrefix.Value,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize Redis session storage")
		}
		sessionStorage = redisStorage
		redisClient = redisStorage.GetClient()
		log.Info().Msg("Session storage configured to use Redis")
	} else {
		// Use in-memory storage
		sessionStorage, err = session.NewStorageFromEnv(
			false, // redisEnabled
			"", "", 0,
			"session:",
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize memory session storage")
		}
		log.Debug().Msg("Using default in-memory session storage")
	}

	// Create session Manager and get Fiber session config
	sessionManager := session.NewManager(sessionStorage, sessionConfig)
	fiberConfig := sessionManager.FiberSessionConfig()

	// Set KeyGenerator (not provided by session-kit's FiberSessionConfig)
	fiberConfig.KeyGenerator = utils.UUID

	return fibersession.New(fiberConfig), redisClient
}

// setupHealthChecker creates a health check aggregator with all dependencies
func setupHealthChecker(redisClient *redis.Client) *health.Aggregator {
	healthConfig := health.DefaultConfig().
		WithServiceName("stargate").
		WithTimeout(5 * time.Second)

	aggregator := health.NewAggregator(healthConfig)

	// Herald health check (if enabled)
	if config.HeraldEnabled.ToBool() {
		heraldURL := config.HeraldURL.String()
		if heraldURL != "" {
			aggregator.AddChecker(health.NewHTTPChecker("herald", heraldURL+"/healthz").
				WithTimeout(2 * time.Second))
		} else {
			aggregator.AddChecker(health.NewDisabledChecker("herald").
				WithMessage("Herald URL not configured"))
		}
	} else {
		aggregator.AddChecker(health.NewDisabledChecker("herald").
			WithMessage("Herald is disabled"))
	}

	// Warden health check (if enabled)
	if config.WardenEnabled.ToBool() {
		wardenURL := config.WardenURL.String()
		if wardenURL != "" {
			aggregator.AddChecker(health.NewHTTPChecker("warden", wardenURL+"/health").
				WithTimeout(2 * time.Second))
		} else {
			aggregator.AddChecker(health.NewDisabledChecker("warden").
				WithMessage("Warden URL not configured"))
		}
	} else {
		aggregator.AddChecker(health.NewDisabledChecker("warden").
			WithMessage("Warden is disabled"))
	}

	// Redis health check (if session storage is enabled)
	if config.SessionStorageEnabled.ToBool() && redisClient != nil {
		aggregator.AddChecker(health.NewRedisChecker(redisClient))
	} else {
		aggregator.AddChecker(health.NewDisabledChecker("redis").
			WithMessage("Session storage is disabled"))
	}

	return aggregator
}

// setupRoutes registers all HTTP routes for the application.
// This includes authentication, login, logout, session exchange, and health check endpoints.
func setupRoutes(app *fiber.App, store *fibersession.Store, healthAggregator *health.Aggregator) {
	log.Debug().Msg("Registering routes")
	// Initialize ForwardAuth handler
	handlers.InitForwardAuthHandler(log)
	// Initialize Herald client
	handlers.InitHeraldClient(log)
	handlers.InitHeraldTOTPClient(log)

	app.Get(RouteHealth, health.FiberHandler(healthAggregator))
	app.Get(RouteRoot, handlers.IndexRoute(store))
	app.Get(RouteLogin, handlers.LoginRoute(store))
	app.Post(RouteLogin, handlers.LoginAPI(store))
	app.Post("/_send_verify_code", handlers.SendVerifyCodeAPI())
	app.Get("/totp/enroll", handlers.TOTPEnrollRoute(store))
	app.Post("/totp/enroll/confirm", handlers.TOTPEnrollConfirmAPI(store))
	app.Get("/totp/revoke", handlers.TOTPRevokeRoute(store))
	app.Post("/totp/revoke", handlers.TOTPRevokeConfirmAPI(store))
	app.Get(RouteLogout, handlers.LogoutRoute(store))
	app.Get(RouteSessionExchange, handlers.SessionShareRoute())
	app.Get(RouteAuth, handlers.CheckRoute(store))
	// Prometheus metrics endpoint
	app.Get("/metrics", metricskit.FiberHandlerFor(metrics.Registry))

	// Register log level endpoint
	logger.RegisterLevelEndpointFiber(app, "/log/level", logger.LevelHandlerConfig{
		Logger:     log,
		AllowedIPs: []string{"127.0.0.1"},
	})
}

// findAssetsPath finds the correct path to assets directory.
func findAssetsPath() string {
	return findFirstExistingPath([]string{
		"./internal/web/templates/assets",
		"./web/templates/assets",
		"./src/internal/web/templates/assets",
	}, "./internal/web/templates/assets")
}

// findFaviconPath finds the correct path to favicon file.
func findFaviconPath() string {
	return findFirstExistingPath([]string{
		"./internal/web/templates/assets/favicon.ico",
		"./web/templates/assets/favicon.ico",
		"./src/internal/web/templates/assets/favicon.ico",
	}, "./internal/web/templates/assets/favicon.ico")
}

// setupStaticFiles registers static file serving for assets.
func setupStaticFiles(app *fiber.App) {
	log.Debug().Msg("Registering static file server for assets")
	assetsPath := findAssetsPath()
	app.Static("/assets", assetsPath)
}

// setupMiddleware configures all middleware for the Fiber application.
// This includes recover, security headers, logging, tracing, i18n, and favicon handling.
func setupMiddleware(app *fiber.App) {
	// 1. Panic recovery (highest priority - prevents server crashes)
	app.Use(recover.New())
	log.Debug().Msg("Panic recovery middleware enabled")

	// 2. Security headers (XSS protection, clickjacking prevention, etc.)
	app.Use(middlewarekit.SecurityHeaders(middlewarekit.DefaultSecurityHeadersConfig()))
	log.Debug().Msg("Security headers middleware enabled")

	// 3. OpenTelemetry tracing middleware (if enabled)
	if config.OTLPEnabled.ToBool() {
		app.Use(internal_tracing.TracingMiddleware("stargate"))
		log.Info().Msg("OpenTelemetry tracing middleware enabled")
	}

	// 4. i18n middleware (language detection from Query > Cookie > Header > Accept-Language)
	app.Use(i18nkit.FiberMiddleware(i18nkit.MiddlewareConfig{
		Bundle: i18n.GetBundle(),
	}))
	log.Debug().Msg("i18n middleware enabled")

	// 5. Request logging with logger-kit
	app.Use(logger.FiberMiddleware(logger.MiddlewareConfig{
		Logger:           log,
		SkipPaths:        []string{"/healthz", "/metrics"},
		IncludeRequestID: true,
		IncludeLatency:   true,
	}))
	log.Debug().Msg("Request logging middleware enabled")

	// 6. Rate limiting (optional - uncomment to enable for production)
	// To enable rate limiting, uncomment the following code:
	//
	// zerologLogger := log.Zerolog()
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
	// log.Info().Msg("Rate limiting middleware enabled")

	// 7. Favicon middleware
	log.Debug().Msg("Adding favicon middleware")
	faviconPath := findFaviconPath()
	// Only add favicon middleware if the file exists
	if _, err := os.Stat(faviconPath); err == nil {
		app.Use(favicon.New(favicon.Config{
			File: faviconPath,
		}))
	} else {
		log.Debug().Str("path", faviconPath).Msg("Favicon file not found, skipping favicon middleware")
	}
}

// createApp creates and configures a new Fiber application.
// It sets up templates, middleware, routes, and static file serving.
//
// Returns a fully configured Fiber app ready to start.
func createApp() *fiber.App {
	engine := setupTemplates()

	log.Debug().Msg("Creating web server instance")
	app := fiber.New(fiber.Config{
		Views:                 engine,
		DisableStartupMessage: true,
	})

	setupMiddleware(app)
	store, redisClient := setupSessionStore()
	healthAggregator := setupHealthChecker(redisClient)

	setupRoutes(app, store, healthAggregator)
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
	if configPort := config.Port.String(); configPort != "" {
		if !strings.HasPrefix(configPort, ":") {
			port = ":" + configPort
		} else {
			port = configPort
		}
		log.Info().Str("port", port).Msg("Using custom port from PORT environment variable")
	}
	log.Debug().Str("port", port).Msg("Starting web server")
	return app.Listen(port)
}
