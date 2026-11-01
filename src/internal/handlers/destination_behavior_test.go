package handlers

import (
	"testing"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func TestVerificationDestinationUsesOnlyCanonicalWardenBindings(t *testing.T) {
	oldSMS, oldEmail := config.LoginSMSEnabled.Value, config.LoginEmailEnabled.Value
	t.Cleanup(func() { config.LoginSMSEnabled.Value = oldSMS; config.LoginEmailEnabled.Value = oldEmail })
	config.LoginSMSEnabled.Value = "true"
	config.LoginEmailEnabled.Value = "true"
	for _, tc := range []struct {
		user                         *warden.AllowListUser
		channel, destination, reason string
	}{
		{nil, "", "", "destination_unavailable"},
		{&warden.AllowListUser{DingtalkUserID: " canonical-id "}, "dingtalk", "canonical-id", ""},
		{&warden.AllowListUser{Phone: " 13800138000 "}, "dingtalk", "13800138000", ""},
		{&warden.AllowListUser{}, "dingtalk", "", "dingtalk_not_bound"},
	} {
		channel, destination, reason := selectVerificationDestination(tc.user, tc.channel)
		require.Equal(t, tc.destination, destination)
		require.Equal(t, tc.reason, reason)
		if reason == "" {
			require.Equal(t, tc.channel, channel)
		}
	}
	ctx, app := createTestContext("POST", "/", nil, "")
	defer app.ReleaseCtx(ctx)
	for _, reason := range []string{"dingtalk_not_bound", "destination_unavailable"} {
		require.NoError(t, verificationSelectionError(ctx, "dingtalk", reason))
		require.Equal(t, 400, ctx.Response().StatusCode())
		require.Contains(t, string(ctx.Response().Body()), reason)
	}
	ctx.Request().Header.Set("Accept-Language", ";q=0.5,en-US")
	require.NotEmpty(t, verificationLocale(ctx))
}
