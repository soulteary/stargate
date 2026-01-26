package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	logger "github.com/soulteary/logger-kit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/tracing-kit"
)

// log is the global logger instance
var log *logger.Logger

// runApplication is the main application logic extracted for testing.
// It performs all initialization steps and starts the server.
// Returns an error if any step fails, allowing tests to verify error handling.
func runApplication() error {
	// Display startup banner
	showBanner()

	// Initialize logger using logger-kit
	initLogger()

	// Initialize configuration
	if err := initConfig(); err != nil {
		return err
	}

	// Initialize OpenTelemetry tracing if enabled
	if config.OTLPEnabled.ToBool() {
		_, err := tracing.InitTracer(
			"stargate",
			Version,
			config.OTLPEndpoint.Value,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize OpenTelemetry tracing")
		} else {
			log.Info().Msg("OpenTelemetry tracing initialized")
		}
	}

	// Create and start server
	app := createApp()

	// Setup graceful shutdown for tracer
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- startServer(app)
	}()

	// Wait for server error or shutdown signal
	select {
	case err := <-serverErr:
		if err != nil {
			return err
		}
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received signal, shutting down gracefully...")

		// Shutdown tracer
		if config.OTLPEnabled.ToBool() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracing.Shutdown(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to shutdown tracer")
			}
		}

		log.Info().Msg("Stargate service stopped")
	}

	return nil
}

// runApplicationWithApp allows injecting a custom app for testing.
// This is useful for testing the application flow without actually starting a server.
func runApplicationWithApp(app *fiber.App) error {
	// Display startup banner
	showBanner()

	// Initialize logger
	initLogger()

	// Initialize configuration
	if err := initConfig(); err != nil {
		return err
	}

	// Start server with provided app
	if err := startServer(app); err != nil {
		return err
	}

	return nil
}

func main() {
	// Use runApplication to handle all initialization and server startup
	// This allows the same logic to be tested via runApplication()
	if err := runApplication(); err != nil {
		log.Fatal().Err(err).Msg("Application failed to start")
	}
}

// showBanner displays the startup banner
func showBanner() {
	pterm.DefaultBox.Println(
		putils.CenterText(
			"Stargate\n" +
				"Your Gateway to Secure Microservices\n" +
				"Version: " + Version,
		),
	)
	time.Sleep(time.Millisecond) // Don't ask why, but this fixes the docker-compose log
}

// initLogger initializes the logging system using logger-kit
func initLogger() {
	log = logger.New(logger.Config{
		Level:          logger.ParseLevelFromEnv("LOG_LEVEL", logger.InfoLevel),
		Format:         logger.FormatJSON,
		ServiceName:    "stargate",
		ServiceVersion: Version,
	})
}

// initConfig initializes the configuration
func initConfig() error {
	if err := config.Initialize(log); err != nil {
		return err
	}

	if config.Debug.ToBool() {
		log.SetLevel(logger.DebugLevel)
	}

	// Initialize Warden client after configuration is loaded
	auth.InitWardenClient(log)

	return nil
}

// GetLogger returns the global logger instance (for use by other packages)
func GetLogger() *logger.Logger {
	return log
}
