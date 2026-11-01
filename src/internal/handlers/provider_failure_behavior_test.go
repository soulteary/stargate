package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestChallengeVerificationProviderFailuresRemainUnauthorized(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		response *herald.VerifyChallengeResponse
		otp      bool
		status   int
	}{
		{"connection with OTP", &herald.HeraldError{Reason: "connection_failed"}, nil, true, 503},
		{"connection without OTP", &herald.HeraldError{}, nil, false, 503},
		{"invalid code", &herald.HeraldError{StatusCode: 400}, &herald.VerifyChallengeResponse{Reason: "invalid"}, false, 401},
		{"expired code", &herald.HeraldError{StatusCode: 401}, &herald.VerifyChallengeResponse{Reason: "expired"}, false, 401},
		{"service authentication", &herald.HeraldError{StatusCode: 401}, nil, false, 401},
		{"unexpected failure", errors.New("backend unavailable"), nil, false, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, app := createTestContext("POST", "/_login", nil, "")
			defer app.ReleaseCtx(ctx)
			require.NoError(t, handleChallengeVerificationError(ctx, tc.err, tc.response, challengeContext{}, tc.otp))
			require.Equal(t, tc.status, ctx.Response().StatusCode())
		})
	}
}

func TestChallengeVerificationRejectsIncompleteOrUnavailableProvider(t *testing.T) {
	previousClient, previousEnabled := heraldClient, config.HeraldEnabled.Value
	t.Cleanup(func() { heraldClient = previousClient; config.HeraldEnabled.Value = previousEnabled })
	heraldClient = nil
	for _, tc := range []struct {
		name, enabled, body string
		otp                 bool
		status              int
	}{
		{"disabled", "false", "", false, 500},
		{"OTP fallback", "false", "", true, 400},
		{"incomplete request", "true", "challenge_id=test", false, 400},
		{"missing provider", "true", "challenge_id=test&verify_code=123456", false, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config.HeraldEnabled.Value = tc.enabled
			ctx, app := createTestContext(fiber.MethodPost, "/_login", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, tc.body)
			defer app.ReleaseCtx(ctx)
			identity := &loginIdentity{userID: "user-1"}
			ok, err := identity.verifyChallenge(ctx, context.Background(), challengeContext{}, tc.otp)
			require.NoError(t, err)
			require.False(t, ok)
			require.Equal(t, tc.status, ctx.Response().StatusCode())
		})
	}
}

func TestChallengeCreationDistinguishesUnavailableAndThrottledProvider(t *testing.T) {
	oldOTP := config.HeraldTOTPEnabled.Value
	t.Cleanup(func() { config.HeraldTOTPEnabled.Value = oldOTP })
	for _, tc := range []struct {
		err    error
		otp    string
		status int
		reason string
	}{
		{&herald.HeraldError{}, "true", 503, "connection_failed"},
		{&herald.HeraldError{}, "false", 503, "connection_failed"},
		{&herald.HeraldError{StatusCode: 429}, "false", 429, "rate_limited"},
	} {
		config.HeraldTOTPEnabled.Value = tc.otp
		ctx, app := createTestContext("POST", "/_send_verify_code", nil, "")
		require.NoError(t, handleChallengeCreationError(ctx, tc.err, "user-1", "email", "user@example.com"))
		require.Equal(t, tc.status, ctx.Response().StatusCode())
		require.Contains(t, string(ctx.Response().Body()), tc.reason)
		app.ReleaseCtx(ctx)
	}
}
