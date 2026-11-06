package handlers

import (
	"encoding/json"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/stretchr/testify/require"
)

// Both successful HTTP responses and provider 4xx responses carry verification
// reasons. They must retain the same translated feedback after extraction.
func TestVerificationFailureMessages(t *testing.T) {
	attempts, wait := 2, 30
	for _, tc := range []struct {
		reason, key    string
		attempts, wait *int
	}{
		{reason: "expired", key: "error.verify_code_expired"},
		{reason: "invalid", key: "error.verify_code_invalid"},
		{reason: "invalid", key: "error.verify_code_invalid_with_attempts", attempts: &attempts},
		{reason: "locked", key: "error.verify_code_locked"},
		{reason: "too_many_attempts", key: "error.verify_code_too_many"},
		{reason: "rate_limited", key: "error.verify_code_rate_limited"},
		{reason: "rate_limited", key: "error.verify_code_rate_limited_with_wait", wait: &wait},
		{reason: "send_failed", key: "error.verify_code_send_failed"},
		{reason: "unauthorized", key: "error.verify_code_unauthorized"},
		{reason: "unknown", key: "error.verify_code_failed"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(ctx fiber.Ctx) error {
				response := &herald.VerifyChallengeResponse{RemainingAttempts: tc.attempts, NextResendIn: tc.wait}
				expected := i18n.T(ctx, tc.key)
				if tc.attempts != nil {
					expected = i18n.Tf(ctx, tc.key, *tc.attempts)
				}
				if tc.wait != nil {
					expected = i18n.Tf(ctx, tc.key, *tc.wait)
				}
				require.Equal(t, expected, verificationFailureMessage(ctx, response, tc.reason))
				return ctx.SendStatus(fiber.StatusOK)
			})
			response, err := app.Test(httptest.NewRequest("GET", "/", nil))
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
		})
	}
}

func TestChallengeVerificationPreservesIdentityAndAMR(t *testing.T) {
	previousClient, previousEnabled := heraldClient, config.HeraldEnabled.Value
	config.HeraldEnabled.Value = "true"
	t.Cleanup(func() { heraldClient = previousClient; config.HeraldEnabled.Value = previousEnabled })
	for _, tc := range []struct {
		name, userID string
		ok           bool
		status       int
	}{
		{"matching identity", "user-1", true, fiber.StatusOK},
		{"mismatched identity", "other-user", false, fiber.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v2/otp/verifications", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(herald.VerifyChallengeV2Response{OK: true, UserID: tc.userID, ChallengeID: "challenge-1", Purpose: "login", Channel: "email", AMR: []string{"otp", "email"}}))
			}))
			defer server.Close()
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)
			app := fiber.New()
			app.Post("/", func(ctx fiber.Ctx) error {
				identity := &loginIdentity{userID: "user-1", wardenUserInfo: &warden.AllowListUser{UserID: "user-1", Mail: "alice@example.com"}}
				ok, err := identity.verifyChallenge(ctx, ctx.Context(), challengeContext{ChallengeID: "challenge-1", UserID: "user-1", Channel: "email", Destination: "alice@example.com"}, false)
				require.Equal(t, tc.ok, ok)
				if !ok {
					return err
				}
				store := session.NewStore()
				sess, getErr := store.Get(ctx)
				require.NoError(t, getErr)
				defer sess.Release()
				storeLoginIdentity(sess, "warden", identity)
				require.Equal(t, []string{"otp", "email"}, sess.Get("user_amr"))
				return ctx.SendStatus(fiber.StatusOK)
			})
			request := httptest.NewRequest("POST", "/", strings.NewReader("challenge_id=challenge-1&verify_code=123456"))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err := app.Test(request)
			require.NoError(t, err)
			require.Equal(t, tc.status, response.StatusCode)
			require.NoError(t, response.Body.Close())
		})
	}
}
