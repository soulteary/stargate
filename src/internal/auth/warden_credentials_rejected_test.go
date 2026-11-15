package auth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

const credentialsRejectedLogFragment = "Warden rejected this Stargate instance's own credentials"

// A 401 from Warden is about Stargate's API key or HMAC signature, not about
// the user. It must not be classified as a missing identity, and the log line
// must name the settings to check without carrying their values.
func TestLookupUserInfoReportsRejectedWardenCredentials(t *testing.T) {
	isolateGlobalState(t)
	previousClient, previousLog := wardenClient, log
	t.Cleanup(func() { wardenClient, log = previousClient, previousLog })

	const (
		apiKey   = "stargate-api-key-value"
		keyID    = "stargate-key-id-value"
		secret   = "stargate-hmac-secret-value"
		audience = "warden-audience-value"
	)
	tests := []struct {
		name       string
		status     int
		hmac       bool
		wantReject bool
		wantKind   string
	}{
		{name: "api key refused", status: http.StatusUnauthorized, wantReject: true, wantKind: `"credential":"api_key"`},
		{name: "hmac signature refused", status: http.StatusUnauthorized, hmac: true, wantReject: true, wantKind: `"credential":"hmac"`},
		{name: "user not listed", status: http.StatusNotFound},
		{name: "warden server error", status: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			config.WardenEnabled.Value = "true"
			config.WardenAPIKey.Value = apiKey
			config.WardenHMACKeyID.Value, config.WardenHMACSecret.Value, config.WardenHMACAudience.Value = "", "", ""
			opts := warden.DefaultOptions().WithBaseURL(server.URL).WithAPIKey(apiKey)
			if tt.hmac {
				config.WardenHMACKeyID.Value, config.WardenHMACSecret.Value, config.WardenHMACAudience.Value = keyID, secret, audience
				opts = opts.WithHMAC(keyID, secret).WithHMACAudience(audience)
			}
			client, err := warden.NewClient(opts)
			require.NoError(t, err)
			wardenClient = client

			var output bytes.Buffer
			log = logger.New(logger.Config{Level: logger.InfoLevel, Format: logger.FormatJSON, Output: &output})

			// Both identifiers are supplied: a refused credential must not
			// trigger the mail fallback that a missing phone number does.
			user, deadline, err := LookupUserInfo(context.Background(), "13800138000", "user@example.com")
			require.Nil(t, user, "a failed lookup never returns an identity")
			require.True(t, deadline.IsZero())
			require.Error(t, err)

			logged := output.String()
			if !tt.wantReject {
				require.NotErrorIs(t, err, ErrWardenCredentialsRejected)
				require.NotContains(t, logged, credentialsRejectedLogFragment)
				require.Equal(t, tt.status == http.StatusNotFound, err == ErrUserUnavailable)
				return
			}
			require.ErrorIs(t, err, ErrWardenCredentialsRejected)
			require.NotErrorIs(t, err, ErrUserUnavailable, "a refused credential is not a missing or inactive user")
			require.Equal(t, int32(1), requests.Load(), "no mail fallback after a refused credential")
			require.Contains(t, logged, `"level":"error"`)
			require.Contains(t, logged, credentialsRejectedLogFragment)
			require.Contains(t, logged, tt.wantKind)
			for _, name := range []string{"WARDEN_API_KEY", "WARDEN_HMAC_KEY_ID", "WARDEN_HMAC_SECRET", "WARDEN_HMAC_AUDIENCE", "WARDEN_HMAC_AUDIENCES"} {
				require.Contains(t, logged, name)
			}
			for _, value := range []string{apiKey, keyID, secret, audience, "13800138000", "user@example.com"} {
				require.NotContains(t, logged, value, "the log line carries no credential or identifier value")
			}
			require.Nil(t, GetUserInfo(context.Background(), "13800138000", ""), "GetUserInfo stays fail-closed")
		})
	}
}
