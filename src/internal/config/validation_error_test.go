package config

import (
	"errors"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func initializeForValidationError(t *testing.T, values map[string]string) *ValidationError {
	t.Helper()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	for name, value := range values {
		t.Setenv(name, value)
	}
	err := Initialize(testLogger())
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Initialize returned %v, want a *ValidationError", err)
	}
	return validation
}

// A rule that involves another setting is an explanation. It must not be
// printed as if it were the value the operator configured, and the accepted
// values of the setting itself say nothing about it.
func TestCrossFieldValidationErrorIsNotRenderedAsProvidedValue(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		key    string
		reason string
	}{
		{"Herald without Warden", map[string]string{"HERALD_ENABLED": "true", "HERALD_URL": "https://herald.example"}, "HERALD_ENABLED", "requires WARDEN_ENABLED=true to resolve authenticated users"},
		{"header auth without Warden", map[string]string{"HEADER_AUTH_ENABLED": "true"}, "HEADER_AUTH_ENABLED", "requires WARDEN_ENABLED=true"},
		{"TOTP without Herald", map[string]string{"HERALD_TOTP_ENABLED": "true"}, "HERALD_TOTP_ENABLED", "requires HERALD_ENABLED=true"},
		{"step-up without proxies", map[string]string{"STEP_UP_ENABLED": "true", "STEP_UP_PATHS": "/admin"}, "TRUSTED_PROXIES", "must identify the proxies allowed to provide the original request URI when step-up is enabled"},
		{"Warden HMAC half configured", map[string]string{"WARDEN_HMAC_KEY_ID": "stargate"}, "WARDEN_HMAC_SECRET", "HMAC key ID and secret must be configured together"},
		{"port out of range", map[string]string{"PORT": "70000"}, "PORT", "must be a TCP port between 1 and 65535"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := initializeForValidationError(t, tc.values)

			testza.AssertEqual(t, tc.key, err.KeyName)
			testza.AssertEqual(t, tc.reason, err.Reason)
			testza.AssertEqual(t, "", err.ProvidedValue)
			testza.AssertFalse(t, err.Missing)

			message := err.Error()
			testza.AssertContains(t, message, "environment variable '"+tc.key+"': "+tc.reason+".")
			testza.AssertNotContains(t, message, "invalid value")
			testza.AssertNotContains(t, message, "'"+tc.reason+"'")
			testza.AssertNotContains(t, message, "Accepted values")
			testza.AssertNotContains(t, message, "[true false]")
		})
	}
}

func TestMissingRequiredSettingIsReportedAsMissing(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		key    string
	}{
		{"required variable", map[string]string{"AUTH_HOST": ""}, "AUTH_HOST"},
		{"passwords without Warden", map[string]string{"PASSWORDS": ""}, "PASSWORDS"},
		{"Warden URL", map[string]string{"WARDEN_ENABLED": "true"}, "WARDEN_URL"},
		{"header auth secret", map[string]string{"WARDEN_ENABLED": "true", "WARDEN_URL": "https://warden.example", "HEADER_AUTH_ENABLED": "true"}, "HEADER_AUTH_SHARED_SECRET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := initializeForValidationError(t, tc.values)

			testza.AssertEqual(t, tc.key, err.KeyName)
			testza.AssertTrue(t, err.Missing)
			testza.AssertEqual(t, "", err.Reason)
			message := err.Error()
			testza.AssertContains(t, message, "environment variable '"+tc.key+"' is required but not set")
			testza.AssertNotContains(t, message, "invalid value")
		})
	}
}

// A value that is itself not accepted keeps the existing message.
func TestInvalidValueErrorStillShowsValueAndAcceptedValues(t *testing.T) {
	err := initializeForValidationError(t, map[string]string{"DEBUG": "maybe"})

	testza.AssertEqual(t, "DEBUG", err.KeyName)
	testza.AssertEqual(t, "maybe", err.ProvidedValue)
	testza.AssertEqual(t, "", err.Reason)
	testza.AssertFalse(t, err.Missing)
	testza.AssertContains(t, err.Error(), "invalid value for environment variable 'DEBUG': 'maybe'")
	testza.AssertContains(t, err.Error(), "Accepted values: [true false]")
}

func TestRuleErrorNeverContainsTheConfiguredSecret(t *testing.T) {
	err := initializeForValidationError(t, map[string]string{"SESSION_EXCHANGE_SECRET": "too-short-secret"})

	testza.AssertEqual(t, "SESSION_EXCHANGE_SECRET", err.KeyName)
	testza.AssertEqual(t, "must contain at least 32 characters", err.Reason)
	testza.AssertNotContains(t, err.Error(), "too-short-secret")
}
