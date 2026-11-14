package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func setRedisOptionsTestEnv(t *testing.T, values map[string]string) {
	t.Helper()
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	for name, value := range values {
		t.Setenv(name, value)
	}
}

func TestInitializeAcceptsRedisUsernameAndTLSOptions(t *testing.T) {
	setRedisOptionsTestEnv(t, map[string]string{
		"SESSION_STORAGE_ENABLED":                "true",
		"SESSION_STORAGE_REDIS_USERNAME":         "stargate",
		"SESSION_STORAGE_REDIS_PASSWORD":         "redis-secret",
		"SESSION_STORAGE_REDIS_TLS_ENABLED":      "TRUE",
		"SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE": "/run/secrets/redis-ca.pem",
		"SESSION_STORAGE_REDIS_TLS_SERVER_NAME":  "redis.internal.example",
	})

	testza.AssertNoError(t, Initialize(testLogger()))
	testza.AssertEqual(t, "stargate", SessionStorageRedisUsername.Value)
	testza.AssertTrue(t, SessionStorageRedisTLSEnabled.ToBool())
	testza.AssertEqual(t, "/run/secrets/redis-ca.pem", SessionStorageRedisTLSCACertFile.Value)
	testza.AssertEqual(t, "redis.internal.example", SessionStorageRedisTLSServerName.Value)
}

func TestInitializeKeepsRedisOptionsOffByDefault(t *testing.T) {
	setRedisOptionsTestEnv(t, nil)

	testza.AssertNoError(t, Initialize(testLogger()))
	testza.AssertEqual(t, "", SessionStorageRedisUsername.Value)
	testza.AssertFalse(t, SessionStorageRedisTLSEnabled.ToBool())
	testza.AssertEqual(t, "", SessionStorageRedisTLSCACertFile.Value)
	testza.AssertEqual(t, "", SessionStorageRedisTLSServerName.Value)
}

func TestInitializeRejectsInconsistentRedisOptions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		want   []string
	}{
		{
			name:   "user name without password",
			values: map[string]string{"SESSION_STORAGE_REDIS_USERNAME": "stargate"},
			want:   []string{"SESSION_STORAGE_REDIS_USERNAME", "requires SESSION_STORAGE_REDIS_PASSWORD"},
		},
		{
			name:   "user name with whitespace",
			values: map[string]string{"SESSION_STORAGE_REDIS_USERNAME": "star gate", "SESSION_STORAGE_REDIS_PASSWORD": "redis-secret"},
			want:   []string{"SESSION_STORAGE_REDIS_USERNAME", "must not contain whitespace"},
		},
		{
			name:   "CA file without TLS",
			values: map[string]string{"SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE": "/run/secrets/redis-ca.pem"},
			want:   []string{"SESSION_STORAGE_REDIS_TLS_CA_CERT_FILE", "requires SESSION_STORAGE_REDIS_TLS_ENABLED=true"},
		},
		{
			name:   "server name without TLS",
			values: map[string]string{"SESSION_STORAGE_REDIS_TLS_ENABLED": "false", "SESSION_STORAGE_REDIS_TLS_SERVER_NAME": "redis.internal.example"},
			want:   []string{"SESSION_STORAGE_REDIS_TLS_SERVER_NAME", "requires SESSION_STORAGE_REDIS_TLS_ENABLED=true"},
		},
		{
			name:   "server name that is not a host",
			values: map[string]string{"SESSION_STORAGE_REDIS_TLS_ENABLED": "true", "SESSION_STORAGE_REDIS_TLS_SERVER_NAME": "redis.internal.example:6380"},
			want:   []string{"SESSION_STORAGE_REDIS_TLS_SERVER_NAME", "must be a DNS name or IP address"},
		},
		{
			name:   "TLS switch that is not a boolean",
			values: map[string]string{"SESSION_STORAGE_REDIS_TLS_ENABLED": "yes"},
			want:   []string{"SESSION_STORAGE_REDIS_TLS_ENABLED"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setRedisOptionsTestEnv(t, tc.values)

			err := Initialize(testLogger())
			testza.AssertNotNil(t, err)
			if err == nil {
				return
			}
			for _, want := range tc.want {
				testza.AssertContains(t, err.Error(), want)
			}
			testza.AssertNotContains(t, err.Error(), "redis-secret")
		})
	}
}

func TestInitializeAcceptsIPAddressAsRedisTLSServerName(t *testing.T) {
	setRedisOptionsTestEnv(t, map[string]string{
		"SESSION_STORAGE_REDIS_TLS_ENABLED":     "true",
		"SESSION_STORAGE_REDIS_TLS_SERVER_NAME": "10.0.0.5",
	})

	testza.AssertNoError(t, Initialize(testLogger()))
}
