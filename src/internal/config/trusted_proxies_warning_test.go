package config

import (
	"bytes"
	"testing"

	"github.com/MarvinJWendt/testza"
	logger "github.com/soulteary/logger-kit/v3"
)

func initializeForTrustedProxiesWarning(t *testing.T, values map[string]string) string {
	t.Helper()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	for name, value := range values {
		t.Setenv(name, value)
	}
	var output bytes.Buffer
	log := logger.New(logger.Config{Level: logger.WarnLevel, Format: logger.FormatJSON, Output: &output})
	testza.AssertNoError(t, Initialize(log))
	return output.String()
}

// The warning has to name what an operator actually sees behind a proxy:
// users are not sent back to the application, and the redirect uses http://.
func TestInitializeWarnsWhatEmptyTrustedProxiesBreaks(t *testing.T) {
	output := initializeForTrustedProxiesWarning(t, nil)

	testza.AssertContains(t, output, "TRUSTED_PROXIES is empty")
	for _, effect := range []string{"loses its callback", "uses http://", "not returned to the application", "one rate-limit quota"} {
		testza.AssertContains(t, output, effect)
	}
}

// Redirects break whether or not rate limits are enabled.
func TestInitializeWarnsAboutEmptyTrustedProxiesWithRateLimitsDisabled(t *testing.T) {
	output := initializeForTrustedProxiesWarning(t, map[string]string{
		"RATE_LIMIT_LOGIN_MAX":        "0",
		"RATE_LIMIT_VERIFICATION_MAX": "0",
	})

	testza.AssertContains(t, output, "TRUSTED_PROXIES is empty")
	testza.AssertContains(t, output, "loses its callback")
}

func TestInitializeDoesNotWarnWhenTrustedProxiesAreSet(t *testing.T) {
	output := initializeForTrustedProxiesWarning(t, map[string]string{"TRUSTED_PROXIES": "10.0.0.0/8"})

	testza.AssertNotContains(t, output, "TRUSTED_PROXIES is empty")
}
