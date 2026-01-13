package main

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/sirupsen/logrus"
	"github.com/soulteary/stargate/src/internal/config"
)

func TestShowBanner(t *testing.T) {
	// Test that showBanner doesn't panic
	// Since it uses pterm which outputs to stdout, we can't easily verify output
	// But we can ensure it doesn't crash
	testza.AssertNotPanics(t, func() {
		showBanner()
	})
}

func TestInitLogger(t *testing.T) {
	// Test that initLogger sets up the logger correctly
	// Save original formatter
	originalFormatter := logrus.StandardLogger().Formatter

	// Call initLogger
	initLogger()

	// Verify formatter is set (should be TextFormatter)
	testza.AssertNotNil(t, logrus.StandardLogger().Formatter)

	// Restore original formatter
	logrus.SetFormatter(originalFormatter)
}

func TestInitConfig_Success(t *testing.T) {
	// Reset log level before test
	logrus.SetLevel(logrus.InfoLevel)

	// Setup required environment variables
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")

	// Reset config state by reinitializing
	err := initConfig()
	testza.AssertNoError(t, err)

	// Verify debug level is set correctly
	testza.AssertEqual(t, logrus.InfoLevel, logrus.GetLevel())
}

func TestInitConfig_WithDebug(t *testing.T) {
	// Reset log level before test
	logrus.SetLevel(logrus.InfoLevel)

	// Setup required environment variables
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "true")

	// Reset config state by reinitializing
	err := initConfig()
	testza.AssertNoError(t, err)

	// Verify debug level is set when DEBUG=true
	testza.AssertEqual(t, logrus.DebugLevel, logrus.GetLevel())
}

func TestInitConfig_ConfigInitializationError(t *testing.T) {
	// Setup invalid environment variables to cause initialization error
	t.Setenv("AUTH_HOST", "")
	t.Setenv("PASSWORDS", "")

	// Reset config state
	config.Initialize()

	// Call initConfig - should return error
	err := initConfig()
	testza.AssertNotNil(t, err)
}

func TestInitConfig_MissingRequiredConfig(t *testing.T) {
	// Clear required environment variables
	os.Unsetenv("AUTH_HOST")
	os.Unsetenv("PASSWORDS")

	// Reset config state
	config.Initialize()

	// Call initConfig - should return error
	err := initConfig()
	testza.AssertNotNil(t, err)
}

func TestInitConfig_DebugFalse(t *testing.T) {
	// Reset log level before test (important: previous test may have set it to DebugLevel)
	logrus.SetLevel(logrus.InfoLevel)

	// Setup required environment variables with DEBUG=false
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")

	// Reset config state
	err := initConfig()
	testza.AssertNoError(t, err)

	// Verify log level is Info (not Debug)
	// Note: initConfig only sets DebugLevel when DEBUG=true,
	// so when DEBUG=false, it should remain at InfoLevel (default)
	testza.AssertEqual(t, logrus.InfoLevel, logrus.GetLevel())
}

func TestInitConfig_DebugCaseInsensitive(t *testing.T) {
	// Test that DEBUG value is case insensitive
	tests := []struct {
		name     string
		debugVal string
		expected logrus.Level
	}{
		{"uppercase TRUE", "TRUE", logrus.DebugLevel},
		{"lowercase true", "true", logrus.DebugLevel},
		{"mixed case True", "True", logrus.DebugLevel},
		{"uppercase FALSE", "FALSE", logrus.InfoLevel},
		{"lowercase false", "false", logrus.InfoLevel},
		{"mixed case False", "False", logrus.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset log level before each subtest
			logrus.SetLevel(logrus.InfoLevel)

			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", "plaintext:test123")
			t.Setenv("DEBUG", tt.debugVal)

			// Reset config state
			err := initConfig()
			testza.AssertNoError(t, err)

			// Verify log level matches expected
			testza.AssertEqual(t, tt.expected, logrus.GetLevel())
		})
	}
}
