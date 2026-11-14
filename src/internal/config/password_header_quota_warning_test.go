package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

const sharedPasswordHeaderQuotaWarning = "PASSWORD_HEADER_AUTH_ENABLED is true and TRUSTED_PROXIES is empty"

// Without TRUSTED_PROXIES every header client is counted at the proxy address.
// The warning has to say what follows from that: one client can use up the
// failure quota for all of them.
func TestInitializeWarnsAboutSharedPasswordHeaderQuota(t *testing.T) {
	output := initializeForTrustedProxiesWarning(t, map[string]string{"PASSWORD_HEADER_AUTH_ENABLED": "true"})

	testza.AssertContains(t, output, sharedPasswordHeaderQuotaWarning)
	for _, effect := range []string{"shared by every client behind that proxy", "one client", "429", "correct password included", "Set TRUSTED_PROXIES"} {
		testza.AssertContains(t, output, effect)
	}
}

func TestInitializeDoesNotWarnAboutSharedPasswordHeaderQuota(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"header authentication disabled": nil,
		"trusted proxies set":            {"PASSWORD_HEADER_AUTH_ENABLED": "true", "TRUSTED_PROXIES": "10.0.0.0/8"},
		"login quota disabled":           {"PASSWORD_HEADER_AUTH_ENABLED": "true", "RATE_LIMIT_LOGIN_MAX": "0"},
	} {
		t.Run(name, func(t *testing.T) {
			output := initializeForTrustedProxiesWarning(t, values)

			testza.AssertNotContains(t, output, sharedPasswordHeaderQuotaWarning)
		})
	}
}
