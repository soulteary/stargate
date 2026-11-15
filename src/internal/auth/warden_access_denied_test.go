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

// A 403 from Warden is about this Stargate instance (its address or the
// actions granted to its key), not about the user. It must not be classified
// as a missing identity or as a refused credential, and the log line must name
// the Warden settings to check without carrying a credential or identifier.
func TestLookupUserInfoReportsWardenAccessDenied(t *testing.T) {
	isolateGlobalState(t)
	previousClient, previousLog := wardenClient, log
	t.Cleanup(func() { wardenClient, log = previousClient, previousLog })

	const (
		keyID  = "stargate-key-id-value"
		secret = "stargate-hmac-secret-value"
	)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		// The body Warden's IP allow list and service policy checks send.
		http.Error(w, "Forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	config.WardenEnabled.Value = "true"
	config.WardenAPIKey.Value = ""
	config.WardenHMACKeyID.Value, config.WardenHMACSecret.Value, config.WardenHMACAudience.Value = keyID, secret, ""
	client, err := warden.NewClient(warden.DefaultOptions().WithBaseURL(server.URL).WithHMAC(keyID, secret))
	require.NoError(t, err)
	wardenClient = client

	var output bytes.Buffer
	log = logger.New(logger.Config{Level: logger.InfoLevel, Format: logger.FormatJSON, Output: &output})

	user, deadline, err := LookupUserInfo(context.Background(), "13800138000", "user@example.com")
	require.Nil(t, user, "a failed lookup never returns an identity")
	require.True(t, deadline.IsZero())
	require.ErrorIs(t, err, ErrWardenAccessDenied)
	require.NotErrorIs(t, err, ErrUserUnavailable, "a refused client is not a missing or inactive user")
	require.NotErrorIs(t, err, ErrWardenCredentialsRejected)
	require.Equal(t, WardenFailureAccessDenied, WardenLookupFailureClass(err))
	require.Equal(t, int32(1), requests.Load(), "no mail fallback after a refused request")

	logged := output.String()
	require.Contains(t, logged, `"level":"error"`)
	require.Contains(t, logged, "Warden refused this Stargate instance with HTTP 403")
	require.Contains(t, logged, `"credential":"hmac"`)
	require.NotContains(t, logged, "rejected this Stargate instance's own credentials")
	for _, name := range []string{"IP_WHITELIST", "TRUSTED_PROXY_IPS", "WARDEN_SERVICE_POLICIES", "directory:lookup", "WARDEN_HMAC_KEY_ID"} {
		require.Contains(t, logged, name)
	}
	for _, value := range []string{keyID, secret, "13800138000", "user@example.com", "127.0.0.1"} {
		require.NotContains(t, logged, value, "the log line carries no credential, address or identifier value")
	}
	require.Nil(t, GetUserInfo(context.Background(), "13800138000", ""), "GetUserInfo stays fail-closed")
}

// Lookup failures without a Warden status are classified too, and none of
// them is an answer about the user.
func TestWardenLookupFailureClass(t *testing.T) {
	isolateGlobalState(t)
	previousClient, previousLog := wardenClient, log
	t.Cleanup(func() { wardenClient, log = previousClient, previousLog })
	log = nil

	require.Empty(t, WardenLookupFailureClass(nil))
	require.Empty(t, WardenLookupFailureClass(ErrUserUnavailable))
	require.Equal(t, WardenFailureUnknown, WardenLookupFailureClass(context.Canceled))

	config.WardenEnabled.Value = "false"
	_, _, err := LookupUserInfo(context.Background(), "13800138000", "")
	require.Equal(t, WardenFailureClientUnavailable, WardenLookupFailureClass(err))
	require.NotErrorIs(t, err, ErrUserUnavailable)

	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	config.WardenEnabled.Value = "true"
	config.WardenSourceMaxAge.Value = "0s"
	client, err := warden.NewClient(warden.DefaultOptions().WithBaseURL(url))
	require.NoError(t, err)
	wardenClient = client
	_, _, err = LookupUserInfo(context.Background(), "13800138000", "")
	require.Equal(t, WardenFailureNoResponse, WardenLookupFailureClass(err))
	require.NotErrorIs(t, err, ErrUserUnavailable)
}
