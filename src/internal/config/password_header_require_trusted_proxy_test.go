package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	logger "github.com/soulteary/logger-kit/v3"
)

func initializeForRequireTrustedProxy(t *testing.T, values map[string]string) error {
	t.Helper()
	isolateGlobalState(t)
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	for name, value := range values {
		t.Setenv(name, value)
	}
	return Initialize(logger.New(logger.Config{Level: logger.ErrorLevel, Format: logger.FormatJSON, Output: testWriter{t}}))
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

// With the opt-in and no trusted proxy, every header request would be refused
// at run time. Startup has to fail and name the setting to fix.
func TestInitializeRefusesRequireTrustedProxyWithoutTrustedProxies(t *testing.T) {
	err := initializeForRequireTrustedProxy(t, map[string]string{
		"PASSWORD_HEADER_AUTH_ENABLED":          "true",
		"PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY": "true",
	})

	if err == nil {
		t.Fatal("Initialize accepted the configuration")
	}
	testza.AssertContains(t, err.Error(), "TRUSTED_PROXIES")
	testza.AssertContains(t, err.Error(), "PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY")
}

func TestInitializeAcceptsRequireTrustedProxy(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"trusted proxies set": {
			"PASSWORD_HEADER_AUTH_ENABLED":          "true",
			"PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY": "true",
			"TRUSTED_PROXIES":                       "10.0.0.0/8",
		},
		"header authentication disabled": {"PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY": "true"},
		"opt-in unset":                   {"PASSWORD_HEADER_AUTH_ENABLED": "true"},
		"opt-in false":                   {"PASSWORD_HEADER_AUTH_ENABLED": "true", "PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY": "false"},
	} {
		t.Run(name, func(t *testing.T) {
			testza.AssertNoError(t, initializeForRequireTrustedProxy(t, values))
		})
	}
}

func TestInitializeRejectsInvalidRequireTrustedProxy(t *testing.T) {
	err := initializeForRequireTrustedProxy(t, map[string]string{"PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY": "maybe"})

	if err == nil {
		t.Fatal("Initialize accepted the configuration")
	}
	testza.AssertContains(t, err.Error(), "PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY")
}
