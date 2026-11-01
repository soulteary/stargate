package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func TestWardenLookupFailsClosedWithoutClientOrOnProviderFailure(t *testing.T) {
	previousClient, previousEnabled := wardenClient, config.WardenEnabled.Value
	t.Cleanup(func() { wardenClient = previousClient; config.WardenEnabled.Value = previousEnabled })
	log = testLogger()
	config.WardenEnabled.Value = "true"
	wardenClient = nil
	require.False(t, CheckUserInList(context.Background(), "13800138000", ""))
	require.Nil(t, GetUserInfo(context.Background(), "13800138000", ""))
	for _, status := range []int{400, 404} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
		var err error
		wardenClient, err = warden.NewClient(warden.DefaultOptions().WithBaseURL(server.URL))
		require.NoError(t, err)
		require.Nil(t, GetUserInfo(context.Background(), "13800138000", ""))
		require.Nil(t, GetUserInfo(context.Background(), "", "missing@example.com"))
		require.Nil(t, GetUserInfo(context.Background(), "13800138000", "missing@example.com"))
		require.False(t, CheckUserInList(context.Background(), "", ""))
		require.Nil(t, GetUserInfo(context.Background(), "", ""))
		server.Close()
	}
}

func TestInvalidPasswordConfigurationAndOTPFailClosed(t *testing.T) {
	previous := config.Passwords.Value
	t.Cleanup(func() { config.Passwords.Value = previous })
	log = testLogger()
	config.Passwords.Value = "invalid"
	algorithm, passwords := GetValidPasswords()
	require.Empty(t, algorithm)
	require.Empty(t, passwords)
	require.False(t, CheckPassword("password"))
	require.False(t, VerifyOTP("JBSWY3DPEHPK3PXP", "abcdef"))
}

func TestWardenInitializationRequiresURLAndRetainsTLSSettings(t *testing.T) {
	variables := []*config.EnvVariable{&config.WardenEnabled, &config.WardenURL, &config.WardenHMACKeyID, &config.WardenHMACSecret, &config.WardenTLSServerName}
	old := make([]string, len(variables))
	for i, v := range variables {
		old[i] = v.Value
	}
	t.Cleanup(func() {
		ResetWardenClientForTesting()
		for i, v := range variables {
			v.Value = old[i]
		}
	})
	for _, v := range variables {
		v.Value = ""
	}
	config.WardenEnabled.Value = "true"
	ResetWardenClientForTesting()
	require.Error(t, InitWardenClient(testLogger()))
	require.Nil(t, getWardenClient())
	ResetWardenClientForTesting()
	config.WardenURL.Value = "https://warden.example.com"
	config.WardenTLSServerName.Value = "warden.example.com"
	config.WardenHMACKeyID.Value = "primary"
	config.WardenHMACSecret.Value = "test-hmac-secret"
	require.NoError(t, InitWardenClient(testLogger()))
	require.NotNil(t, getWardenClient())
}
