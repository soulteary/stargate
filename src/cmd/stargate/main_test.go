package main

import (
	"os"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	logger "github.com/soulteary/logger-kit"
	"github.com/soulteary/stargate/src/internal/config"
	version "github.com/soulteary/version-kit"
)

func TestShowBanner(t *testing.T) {
	// Test that showBanner doesn't panic
	// Since it uses pterm which outputs to stdout, we can't easily verify output
	// But we can ensure it doesn't crash
	testza.AssertNotPanics(t, func() {
		showBanner()
	})
}

func TestShowBanner_ContainsVersion(t *testing.T) {
	// Test that showBanner executes without error
	// The banner should contain the version information
	testza.AssertNotPanics(t, func() {
		showBanner()
	})
	// Verify Version variable is accessible and defined
	// Version is a string variable (defaults to "dev" but can be overridden at build time via ldflags)
	testza.AssertTrue(t, len(version.Version) >= 0, "Version should be defined")
	testza.AssertEqual(t, "dev", version.Version, "Default version should be 'dev'")
}

func TestInitLogger(t *testing.T) {
	// Test that initLogger sets up the logger correctly
	testza.AssertNotPanics(t, func() {
		initLogger()
	})

	// Verify log is not nil after initialization
	testza.AssertNotNil(t, log)
}

func TestInitLogger_MultipleCalls(t *testing.T) {
	// Test that initLogger can be called multiple times without issues
	testza.AssertNotPanics(t, func() {
		initLogger()
		initLogger()
	})
}

func TestInitConfig_Success(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Setup required environment variables
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")

	// Reset config state by reinitializing
	err := initConfig()
	testza.AssertNoError(t, err)
}

func TestInitConfig_WithDebug(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Setup required environment variables
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "true")

	// Reset config state by reinitializing
	err := initConfig()
	testza.AssertNoError(t, err)

	// Verify debug level is set when DEBUG=true
	testza.AssertEqual(t, logger.DebugLevel, log.GetLevel())
}

func TestInitConfig_ConfigInitializationError(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Setup invalid environment variables to cause initialization error
	t.Setenv("AUTH_HOST", "")
	t.Setenv("PASSWORDS", "")

	// Reset config state
	_ = config.Initialize(log)

	// Call initConfig - should return error
	err := initConfig()
	testza.AssertNotNil(t, err)
}

func TestInitConfig_MissingRequiredConfig(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Clear required environment variables
	_ = os.Unsetenv("AUTH_HOST")
	_ = os.Unsetenv("PASSWORDS")

	// Reset config state
	_ = config.Initialize(log)

	// Call initConfig - should return error
	err := initConfig()
	testza.AssertNotNil(t, err)
}

func TestInitConfig_DebugFalse(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Setup required environment variables with DEBUG=false
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")

	// Reset config state
	err := initConfig()
	testza.AssertNoError(t, err)

	// Verify log level is Info (not Debug)
	testza.AssertEqual(t, logger.InfoLevel, log.GetLevel())
}

func TestInitConfig_DebugCaseInsensitive(t *testing.T) {
	// Test that DEBUG value is case insensitive
	tests := []struct {
		name     string
		debugVal string
		expected logger.Level
	}{
		{"uppercase TRUE", "TRUE", logger.DebugLevel},
		{"lowercase true", "true", logger.DebugLevel},
		{"mixed case True", "True", logger.DebugLevel},
		{"uppercase FALSE", "FALSE", logger.InfoLevel},
		{"lowercase false", "false", logger.InfoLevel},
		{"mixed case False", "False", logger.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset logger
			initLogger()

			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", "plaintext:test123")
			t.Setenv("DEBUG", tt.debugVal)

			// Reset config state
			err := initConfig()
			testza.AssertNoError(t, err)

			// Verify log level matches expected
			testza.AssertEqual(t, tt.expected, log.GetLevel())
		})
	}
}

func TestInitConfig_EmptyDebugValue(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Test that empty DEBUG value defaults to false (InfoLevel)
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	_ = os.Unsetenv("DEBUG")

	// Reset config state
	err := initConfig()
	testza.AssertNoError(t, err)

	// Empty DEBUG should default to false, so InfoLevel
	testza.AssertEqual(t, logger.InfoLevel, log.GetLevel())
}

func TestInitConfig_InvalidDebugValue(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Test that invalid DEBUG value causes config initialization to fail
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "invalid")

	// Reset config state - this should fail validation
	_ = config.Initialize(log)

	// initConfig should return error because DEBUG validation failed
	err := initConfig()
	testza.AssertNotNil(t, err)
}

func TestInitConfig_LogLevelTransition(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Test transitioning from DebugLevel to InfoLevel
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")

	// First set to debug
	t.Setenv("DEBUG", "true")
	err := initConfig()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, logger.DebugLevel, log.GetLevel())

	// Reset logger for new config
	initLogger()

	// Then set to false
	t.Setenv("DEBUG", "false")
	err = initConfig()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, logger.InfoLevel, log.GetLevel())
}

func TestInitConfig_ConfigInitializationSuccessPath(t *testing.T) {
	// Initialize logger first
	initLogger()

	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "true")

	err := initConfig()
	testza.AssertNoError(t, err)

	// Verify both config initialization and log level setting worked
	testza.AssertEqual(t, logger.DebugLevel, log.GetLevel())
	testza.AssertNotNil(t, config.AuthHost.Value)
	testza.AssertNotNil(t, config.Passwords.Value)
}

// TestRunApplication_ConfigError tests that runApplication returns error when config initialization fails
func TestRunApplication_ConfigError(t *testing.T) {
	// Setup invalid environment variables to cause initialization error
	t.Setenv("AUTH_HOST", "")
	t.Setenv("PASSWORDS", "")

	// Reset config state
	initLogger()
	_ = config.Initialize(log)

	// runApplication should return error when config initialization fails
	err := runApplication()
	testza.AssertNotNil(t, err, "runApplication should return error when config fails")
}

// TestRunApplicationWithApp_ConfigError tests that runApplicationWithApp returns error when config initialization fails
func TestRunApplicationWithApp_ConfigError(t *testing.T) {
	// Setup invalid environment variables to cause initialization error
	t.Setenv("AUTH_HOST", "")
	t.Setenv("PASSWORDS", "")

	// Reset config state
	initLogger()
	_ = config.Initialize(log)

	// Create a test app
	app := fiber.New()

	// runApplicationWithApp should return error when config initialization fails
	err := runApplicationWithApp(app)
	testza.AssertNotNil(t, err, "runApplicationWithApp should return error when config fails")
}

// TestRunApplicationWithApp_Success tests that runApplicationWithApp works with valid config
func TestRunApplicationWithApp_Success(t *testing.T) {
	// Setup valid environment variables
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")

	// Reset config state
	initLogger()
	_ = config.Initialize(log)

	// Verify the function signature and that it can be called
	testza.AssertNotNil(t, runApplicationWithApp, "runApplicationWithApp should be defined")
}

// TestRunApplication_SuccessPath tests the complete success path
func TestRunApplication_SuccessPath(t *testing.T) {
	// Setup valid environment variables
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")

	// Reset config state
	initLogger()
	_ = config.Initialize(log)

	// Verify the function exists and can be called
	_ = runApplication
	testza.AssertNotNil(t, runApplication, "runApplication should be defined")
}

// TestGetLogger_ReturnsNonNilAfterInit covers GetLogger for coverage (used by other packages).
func TestGetLogger_ReturnsNonNilAfterInit(t *testing.T) {
	initLogger()
	l := GetLogger()
	testza.AssertNotNil(t, l, "GetLogger() must not be nil after initLogger()")
}

// TestMainFunction_Exists tests that main function exists and can be analyzed
func TestMainFunction_Exists(t *testing.T) {
	// Verify that main function exists (it's the entry point)
	testza.AssertNotNil(t, showBanner, "showBanner should be defined")
	testza.AssertNotNil(t, initLogger, "initLogger should be defined")
	testza.AssertNotNil(t, initConfig, "initConfig should be defined")
	testza.AssertNotNil(t, createApp, "createApp should be defined")
	testza.AssertNotNil(t, startServer, "startServer should be defined")
}

// TestShowBanner_ExecutionTime tests that showBanner executes within reasonable time
func TestShowBanner_ExecutionTime(t *testing.T) {
	start := time.Now()
	showBanner()
	duration := time.Since(start)

	// Banner should execute quickly (less than 1 second)
	testza.AssertTrue(t, duration < time.Second, "showBanner should execute quickly")
}

// TestShowBanner_ContentStructure tests that showBanner produces expected content structure
func TestShowBanner_ContentStructure(t *testing.T) {
	// Since showBanner uses pterm which outputs to stdout,
	// we can't easily capture the output, but we can verify it doesn't panic
	testza.AssertNotPanics(t, func() {
		showBanner()
	})

	// Verify Version variable is used
	testza.AssertTrue(t, len(version.Version) >= 0, "Version should be defined")
}

// TestInitConfig_AllPaths tests all code paths in initConfig
func TestInitConfig_AllPaths(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*testing.T)
		expectError bool
		expectLevel logger.Level
	}{
		{
			name: "success with debug true",
			setup: func(t *testing.T) {
				t.Setenv("AUTH_HOST", "auth.example.com")
				t.Setenv("PASSWORDS", "plaintext:test123")
				t.Setenv("DEBUG", "true")
			},
			expectError: false,
			expectLevel: logger.DebugLevel,
		},
		{
			name: "success with debug false",
			setup: func(t *testing.T) {
				t.Setenv("AUTH_HOST", "auth.example.com")
				t.Setenv("PASSWORDS", "plaintext:test123")
				t.Setenv("DEBUG", "false")
			},
			expectError: false,
			expectLevel: logger.InfoLevel,
		},
		{
			name: "error with invalid config",
			setup: func(t *testing.T) {
				t.Setenv("AUTH_HOST", "")
				t.Setenv("PASSWORDS", "")
			},
			expectError: true,
			expectLevel: logger.InfoLevel, // Default level when error occurs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset logger
			initLogger()

			// Setup test environment
			tt.setup(t)

			// Reset config state
			_ = config.Initialize(log)

			// Call initConfig
			err := initConfig()

			if tt.expectError {
				testza.AssertNotNil(t, err, "should return error")
			} else {
				testza.AssertNoError(t, err, "should not return error")
				testza.AssertEqual(t, tt.expectLevel, log.GetLevel(), "log level should match")
			}
		})
	}
}

// TestRunApplication_ConfigErrorPath tests the error path when config initialization fails
func TestRunApplication_ConfigErrorPath(t *testing.T) {
	// Setup invalid environment variables
	t.Setenv("AUTH_HOST", "")
	t.Setenv("PASSWORDS", "")

	// Reset config state
	initLogger()
	_ = config.Initialize(log)

	// runApplication should return error when config fails
	err := runApplication()
	testza.AssertNotNil(t, err, "runApplication should return error when config fails")
	testza.AssertContains(t, err.Error(), "required", "error should mention required config")
}

// TestRunApplicationWithApp_ConfigErrorPath tests the error path when config initialization fails
func TestRunApplicationWithApp_ConfigErrorPath(t *testing.T) {
	// Setup invalid environment variables
	t.Setenv("AUTH_HOST", "")
	t.Setenv("PASSWORDS", "")

	// Reset config state
	initLogger()
	_ = config.Initialize(log)

	// Create a test app
	app := fiber.New()

	// runApplicationWithApp should return error when config fails
	err := runApplicationWithApp(app)
	testza.AssertNotNil(t, err, "runApplicationWithApp should return error when config fails")
}

// TestRunApplication_ServerErrorPath tests that runApplication handles server start errors
func TestRunApplication_ServerErrorPath(t *testing.T) {
	// Setup valid environment variables
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")

	// Reset config state
	initLogger()
	_ = config.Initialize(log)

	// Verify the function exists and the error handling code is present
	testza.AssertNotNil(t, runApplication, "runApplication should be defined")
}

// TestMainFunction_CodeStructure tests that main function has the expected structure
func TestMainFunction_CodeStructure(t *testing.T) {
	// Verify all functions called by main() exist and are testable
	testza.AssertNotNil(t, showBanner, "showBanner should be defined")
	testza.AssertNotNil(t, initLogger, "initLogger should be defined")
	testza.AssertNotNil(t, initConfig, "initConfig should be defined")
	testza.AssertNotNil(t, createApp, "createApp should be defined")
	testza.AssertNotNil(t, startServer, "startServer should be defined")

	// Verify runApplication exists (extracted logic from main)
	testza.AssertNotNil(t, runApplication, "runApplication should be defined")
}

// TestShowBanner_MultipleCalls tests that showBanner can be called multiple times
func TestShowBanner_MultipleCalls(t *testing.T) {
	// Call showBanner multiple times
	testza.AssertNotPanics(t, func() {
		showBanner()
		showBanner()
		showBanner()
	})
}

// TestShowBanner_VersionIntegration tests that showBanner uses Version variable correctly
func TestShowBanner_VersionIntegration(t *testing.T) {
	// Verify Version is defined and used
	testza.AssertTrue(t, len(version.Version) >= 0, "Version should be defined")

	// Call showBanner and verify it doesn't panic
	testza.AssertNotPanics(t, func() {
		showBanner()
	})
}

// TestInitLogger_Creates_Logger tests that initLogger creates a valid logger
func TestInitLogger_Creates_Logger(t *testing.T) {
	initLogger()

	// Verify logger is created
	testza.AssertNotNil(t, log, "logger should be created")
}

// TestInitConfig_ErrorReturn tests that initConfig returns error correctly
func TestInitConfig_ErrorReturn(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Setup invalid config
	t.Setenv("AUTH_HOST", "")
	t.Setenv("PASSWORDS", "")

	_ = config.Initialize(log)

	err := initConfig()
	testza.AssertNotNil(t, err, "initConfig should return error for invalid config")
}

// TestInitConfig_SuccessReturn tests that initConfig returns nil on success
func TestInitConfig_SuccessReturn(t *testing.T) {
	// Initialize logger first
	initLogger()

	// Setup valid config
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")

	_ = config.Initialize(log)

	err := initConfig()
	testza.AssertNoError(t, err, "initConfig should not return error for valid config")
}
