package config

import (
	"bytes"
	"testing"

	"github.com/MarvinJWendt/testza"
	logger "github.com/soulteary/logger-kit/v3"
)

func setDemoSecretTestEnv(t *testing.T, cookieSecure string) {
	t.Helper()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("CALLBACK_ALLOWED_HOSTS", "app.example.com")
	t.Setenv("SESSION_EXCHANGE_SECRET", PublishedDemoSessionExchangeSecret)
	if cookieSecure != "" {
		t.Setenv("COOKIE_SECURE", cookieSecure)
	}
}

func TestInitializeRejectsPublishedDemoSecretWhenCookieSecure(t *testing.T) {
	for _, cookieSecure := range []string{"", "true", "TRUE"} {
		t.Run("COOKIE_SECURE="+cookieSecure, func(t *testing.T) {
			setDemoSecretTestEnv(t, cookieSecure)

			err := Initialize(testLogger())
			testza.AssertNotNil(t, err)
			testza.AssertContains(t, err.Error(), "SESSION_EXCHANGE_SECRET")
			testza.AssertContains(t, err.Error(), "published demo value")
		})
	}
}

func TestInitializeAcceptsPublishedDemoSecretForLocalHTTP(t *testing.T) {
	setDemoSecretTestEnv(t, "false")

	testza.AssertNoError(t, Initialize(testLogger()))
}

func TestInitializeAcceptsOtherSecretWhenCookieSecure(t *testing.T) {
	setDemoSecretTestEnv(t, "true")
	t.Setenv("SESSION_EXCHANGE_SECRET", PublishedDemoSessionExchangeSecret+"-rotated")

	testza.AssertNoError(t, Initialize(testLogger()))
}

func TestInitializeWarnsAboutPlaintextPasswordsWhenCookieSecure(t *testing.T) {
	const warning = "PASSWORDS uses the plaintext algorithm while COOKIE_SECURE is true"
	// bcrypt hash of "test123"; only its algorithm prefix matters here.
	const bcryptPasswords = "bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

	for _, tc := range []struct {
		name         string
		passwords    string
		cookieSecure string
		wantWarning  bool
	}{
		{name: "plaintext over HTTPS", passwords: "plaintext:test123", cookieSecure: "true", wantWarning: true},
		{name: "plaintext over local HTTP", passwords: "plaintext:test123", cookieSecure: "false", wantWarning: false},
		{name: "bcrypt over HTTPS", passwords: bcryptPasswords, cookieSecure: "true", wantWarning: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", tc.passwords)
			t.Setenv("COOKIE_SECURE", tc.cookieSecure)
			var output bytes.Buffer
			log := logger.New(logger.Config{Level: logger.InfoLevel, Format: logger.FormatJSON, Output: &output})

			testza.AssertNoError(t, Initialize(log))
			testza.AssertEqual(t, tc.wantWarning, bytes.Contains(output.Bytes(), []byte(warning)))
			testza.AssertFalse(t, bytes.Contains(output.Bytes(), []byte("test123")))
		})
	}
}

func TestInitializeWarnsWhenSharedPasswordCanPassStepUp(t *testing.T) {
	const warning = "sessions created with the shared password still pass step-up"

	for _, tc := range []struct {
		name        string
		herald      bool
		passwords   string
		wantWarning bool
	}{
		{name: "Warden, Herald and shared password", herald: true, passwords: "plaintext:test123", wantWarning: true},
		{name: "Warden and Herald only", herald: true, passwords: "", wantWarning: false},
		{name: "password-only deployment", herald: false, passwords: "plaintext:test123", wantWarning: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", tc.passwords)
			t.Setenv("COOKIE_SECURE", "false")
			t.Setenv("STEP_UP_ENABLED", "true")
			t.Setenv("STEP_UP_PATHS", "/admin")
			t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
			if tc.herald {
				t.Setenv("WARDEN_ENABLED", "true")
				t.Setenv("WARDEN_URL", "http://warden.test")
				t.Setenv("HERALD_ENABLED", "true")
				t.Setenv("HERALD_URL", "http://herald.test")
			}
			var output bytes.Buffer
			log := logger.New(logger.Config{Level: logger.InfoLevel, Format: logger.FormatJSON, Output: &output})

			testza.AssertNoError(t, Initialize(log))
			testza.AssertEqual(t, tc.wantWarning, bytes.Contains(output.Bytes(), []byte(warning)))
		})
	}
}
