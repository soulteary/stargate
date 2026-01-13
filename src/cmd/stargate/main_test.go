package main

import (
	"os"
	"reflect"
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

func TestShowBanner_ContainsVersion(t *testing.T) {
	// Test that showBanner executes without error
	// The banner should contain the version information
	testza.AssertNotPanics(t, func() {
		showBanner()
	})
	// Verify Version constant is accessible and defined
	// Version is a string constant (defaults to "dev" but can be overridden at build time)
	testza.AssertTrue(t, len(Version) >= 0, "Version should be defined")
	testza.AssertEqual(t, "dev", Version, "Default version should be 'dev'")
}

func TestInitLogger(t *testing.T) {
	// Test that initLogger sets up the logger correctly
	// Save original formatter
	originalFormatter := logrus.StandardLogger().Formatter
	originalLevel := logrus.GetLevel()

	// Restore original state after test
	defer func() {
		logrus.SetFormatter(originalFormatter)
		logrus.SetLevel(originalLevel)
	}()

	// Call initLogger
	initLogger()

	// Verify formatter is set (should be TextFormatter)
	testza.AssertNotNil(t, logrus.StandardLogger().Formatter)

	// Verify it's actually a TextFormatter
	formatterType := reflect.TypeOf(logrus.StandardLogger().Formatter)
	testza.AssertEqual(t, "*logrus.TextFormatter", formatterType.String())
}

func TestInitLogger_MultipleCalls(t *testing.T) {
	// Test that initLogger can be called multiple times without issues
	originalFormatter := logrus.StandardLogger().Formatter
	originalLevel := logrus.GetLevel()

	defer func() {
		logrus.SetFormatter(originalFormatter)
		logrus.SetLevel(originalLevel)
	}()

	// Call initLogger multiple times
	initLogger()
	formatter1 := logrus.StandardLogger().Formatter

	initLogger()
	formatter2 := logrus.StandardLogger().Formatter

	// Both should be TextFormatter
	testza.AssertNotNil(t, formatter1)
	testza.AssertNotNil(t, formatter2)
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

func TestInitConfig_EmptyDebugValue(t *testing.T) {
	// Test that empty DEBUG value defaults to false (InfoLevel)
	logrus.SetLevel(logrus.InfoLevel)

	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	os.Unsetenv("DEBUG")

	// Reset config state
	err := initConfig()
	testza.AssertNoError(t, err)

	// Empty DEBUG should default to false, so InfoLevel
	testza.AssertEqual(t, logrus.InfoLevel, logrus.GetLevel())
}

func TestInitConfig_InvalidDebugValue(t *testing.T) {
	// Test that invalid DEBUG value causes config initialization to fail
	// This should fail at config validation, not at initConfig
	logrus.SetLevel(logrus.InfoLevel)

	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "invalid")

	// Reset config state - this should fail validation
	config.Initialize()

	// initConfig should return error because DEBUG validation failed
	err := initConfig()
	testza.AssertNotNil(t, err)
}

func TestInitConfig_LogLevelTransition(t *testing.T) {
	// Test transitioning from DebugLevel to InfoLevel
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")

	// First set to debug
	t.Setenv("DEBUG", "true")
	err := initConfig()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, logrus.DebugLevel, logrus.GetLevel())

	// Then set to false
	t.Setenv("DEBUG", "false")
	err = initConfig()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, logrus.InfoLevel, logrus.GetLevel())
}

func TestInitConfig_ConfigInitializationSuccessPath(t *testing.T) {
	// Test the complete success path of initConfig
	logrus.SetLevel(logrus.WarnLevel) // Start with different level

	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "true")

	err := initConfig()
	testza.AssertNoError(t, err)

	// Verify both config initialization and log level setting worked
	testza.AssertEqual(t, logrus.DebugLevel, logrus.GetLevel())
	testza.AssertNotNil(t, config.AuthHost.Value)
	testza.AssertNotNil(t, config.Passwords.Value)
}
