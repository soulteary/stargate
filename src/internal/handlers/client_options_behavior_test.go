package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/herald/pkg/herald"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestHeraldInitializationValidatesCredentialsAndTLSOptions(t *testing.T) {
	variables := []*config.EnvVariable{&config.HeraldEnabled, &config.HeraldURL, &config.HeraldAPIKey, &config.HeraldHMACSecret, &config.HeraldHMACKeyID, &config.HeraldClientIPHeader, &config.HeraldTLSCACertFile, &config.HeraldTLSClientCert, &config.HeraldTLSClientKey, &config.HeraldTLSServerName}
	old := make([]string, len(variables))
	for i, v := range variables {
		old[i] = v.Value
	}
	t.Cleanup(func() {
		ResetHeraldClientForTest()
		for i, v := range variables {
			v.Value = old[i]
		}
	})
	for _, tc := range []struct {
		name      string
		values    map[*config.EnvVariable]string
		wantError bool
	}{
		{"disabled", nil, false},
		{"missing URL", map[*config.EnvVariable]string{&config.HeraldEnabled: "true"}, true},
		{"HMAC", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldHMACSecret: "test-hmac-secret", &config.HeraldHMACKeyID: "primary"}, false},
		{"client IP header", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldAPIKey: "test-api-key", &config.HeraldClientIPHeader: "X-Forwarded-For"}, false},
		{"reserved client IP header", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldAPIKey: "test-api-key", &config.HeraldClientIPHeader: "X-Signature"}, true},
		{"invalid client IP header", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldAPIKey: "test-api-key", &config.HeraldClientIPHeader: "X Forwarded For"}, true},
		{"certificate without key", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldAPIKey: "test-api-key", &config.HeraldTLSClientCert: "missing.pem"}, false},
		{"TLS options", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldTLSCACertFile: "missing-ca.pem", &config.HeraldTLSClientCert: "missing.pem", &config.HeraldTLSClientKey: "missing.key", &config.HeraldTLSServerName: "herald.example.com"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ResetHeraldClientForTest()
			for _, v := range variables {
				v.Value = ""
			}
			for v, value := range tc.values {
				v.Value = value
			}
			err := InitHeraldClient(testLogger())
			if tc.wantError {
				require.Error(t, err)
				require.Nil(t, getHeraldClient())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHeraldClientForwardsClientIPInConfiguredHeader(t *testing.T) {
	variables := []*config.EnvVariable{&config.HeraldEnabled, &config.HeraldURL, &config.HeraldAPIKey, &config.HeraldHMACSecret, &config.HeraldHMACKeyID, &config.HeraldClientIPHeader, &config.HeraldTLSCACertFile, &config.HeraldTLSClientCert, &config.HeraldTLSClientKey, &config.HeraldTLSServerName}
	old := make([]string, len(variables))
	for i, v := range variables {
		old[i] = v.Value
	}
	t.Cleanup(func() {
		ResetHeraldClientForTest()
		for i, v := range variables {
			v.Value = old[i]
		}
	})
	for _, tc := range []struct {
		name   string
		header string
		want   string
	}{
		{"unset sends no header", "", ""},
		{"set forwards the client IP", "X-Forwarded-For", "203.0.113.7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got <- r.Header.Get("X-Forwarded-For")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"challenge_id":"c1","expires_in":300,"next_resend_in":60}`))
			}))
			defer server.Close()

			ResetHeraldClientForTest()
			for _, v := range variables {
				v.Value = ""
			}
			config.HeraldEnabled.Value = "true"
			config.HeraldURL.Value = server.URL
			config.HeraldAPIKey.Value = "test-api-key"
			config.HeraldClientIPHeader.Value = tc.header
			require.NoError(t, InitHeraldClient(testLogger()))

			_, err := getHeraldClient().CreateChallenge(context.Background(), &herald.CreateChallengeRequest{
				UserID: "u1", Channel: "email", Destination: "user@example.com", Purpose: "login", ClientIP: "203.0.113.7",
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, <-got)
		})
	}
}
