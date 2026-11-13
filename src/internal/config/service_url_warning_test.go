package config

import (
	"bytes"
	"testing"

	"github.com/MarvinJWendt/testza"
	logger "github.com/soulteary/logger-kit/v3"
)

func TestInitializeWarnsAboutPlaintextServiceURLs(t *testing.T) {
	const warning = "Service URL uses http:// while an API key or HMAC secret is configured"

	for _, tc := range []struct {
		name   string
		values map[string]string
		want   []string
	}{
		{
			name:   "Warden API key over http",
			values: map[string]string{"WARDEN_ENABLED": "true", "WARDEN_URL": "http://warden.example:8081", "WARDEN_API_KEY": "warden-api-key-value"},
			want:   []string{`"name":"WARDEN_URL"`},
		},
		{
			name: "Warden and Herald HMAC over http",
			values: map[string]string{
				"WARDEN_ENABLED": "true", "WARDEN_URL": "HTTP://warden.example:8081", "WARDEN_HMAC_KEY_ID": "stargate", "WARDEN_HMAC_SECRET": "warden-hmac-secret-value",
				"HERALD_ENABLED": "true", "HERALD_URL": "http://herald.example:8082", "HERALD_HMAC_SECRET": "herald-hmac-secret-value",
			},
			want: []string{`"name":"WARDEN_URL"`, `"name":"HERALD_URL"`},
		},
		{
			name:   "https",
			values: map[string]string{"WARDEN_ENABLED": "true", "WARDEN_URL": "https://warden.example", "WARDEN_API_KEY": "warden-api-key-value"},
		},
		{
			name:   "http without a credential",
			values: map[string]string{"WARDEN_ENABLED": "true", "WARDEN_URL": "http://warden.example:8081"},
		},
		{
			name:   "disabled integration",
			values: map[string]string{"WARDEN_URL": "http://warden.example:8081", "WARDEN_API_KEY": "warden-api-key-value"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", "plaintext:test123")
			for name, value := range tc.values {
				t.Setenv(name, value)
			}
			var output bytes.Buffer
			log := logger.New(logger.Config{Level: logger.WarnLevel, Format: logger.FormatJSON, Output: &output})

			testza.AssertNoError(t, Initialize(log))
			testza.AssertEqual(t, len(tc.want), bytes.Count(output.Bytes(), []byte(warning)))
			for _, want := range tc.want {
				testza.AssertTrue(t, bytes.Contains(output.Bytes(), []byte(want)), want)
			}
			for _, secret := range []string{"warden-api-key-value", "warden-hmac-secret-value", "herald-hmac-secret-value"} {
				testza.AssertFalse(t, bytes.Contains(output.Bytes(), []byte(secret)))
			}
		})
	}
}
