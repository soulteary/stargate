package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

const ignoredSessionRedisWarning = "Redis session settings are set but SESSION_STORAGE_ENABLED is not true"

// An operator who configures Redis and forgets SESSION_STORAGE_ENABLED gets a
// running service with in-memory sessions. The warning names what was ignored.
func TestInitializeWarnsAboutIgnoredSessionRedisSettings(t *testing.T) {
	output := initializeForTrustedProxiesWarning(t, map[string]string{
		"SESSION_STORAGE_REDIS_ADDR":        "redis.internal:6380",
		"SESSION_STORAGE_REDIS_PASSWORD":    "redis-password-value",
		"SESSION_STORAGE_REDIS_TLS_ENABLED": "true",
	})

	testza.AssertContains(t, output, ignoredSessionRedisWarning)
	testza.AssertContains(t, output, `"ignored":["SESSION_STORAGE_REDIS_ADDR","SESSION_STORAGE_REDIS_PASSWORD","SESSION_STORAGE_REDIS_TLS_ENABLED"]`)
	testza.AssertContains(t, output, "kept in memory")
	testza.AssertNotContains(t, output, "redis-password-value")
}

func TestInitializeDoesNotWarnAboutSessionRedisSettings(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"nothing set": nil,
		"defaults spelled out": {
			"SESSION_STORAGE_REDIS_ADDR":        "localhost:6379",
			"SESSION_STORAGE_REDIS_DB":          "0",
			"SESSION_STORAGE_REDIS_KEY_PREFIX":  "stargate:session:",
			"SESSION_STORAGE_REDIS_TLS_ENABLED": "false",
		},
		"storage enabled": {
			"SESSION_STORAGE_ENABLED":        "true",
			"SESSION_STORAGE_REDIS_ADDR":     "redis.internal:6380",
			"SESSION_STORAGE_REDIS_PASSWORD": "redis-password-value",
		},
	} {
		t.Run(name, func(t *testing.T) {
			output := initializeForTrustedProxiesWarning(t, values)

			testza.AssertNotContains(t, output, ignoredSessionRedisWarning)
		})
	}
}
