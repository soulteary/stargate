package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSendVerifyCodeRetryMetadataHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, body, header string
		want               int
	}{
		{"known", `{"reason":"rate_limited","next_resend_in":37}`, "37", 37},
		{"old backend", `{"reason":"rate_limited"}`, "", 0},
		{"conflicting", `{"reason":"rate_limited","next_resend_in":37}`, "38", 0},
		{"invalid", `{"reason":"rate_limited","next_resend_in":86401}`, "86401", 0},
		{"malformed", `{"reason":"rate_limited","next_resend_in":"37"}`, "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupSendVerifyCodeBaseEnv(t)
			t.Setenv("HERALD_ENABLED", "true")
			t.Setenv("HERALD_HMAC_SECRET", "synthetic-signing-secret")
			t.Setenv("HERALD_HMAC_KEY_ID", "gateway")
			t.Setenv("HERALD_API_KEY", "unused-api-key")
			warden := newWardenUserServer(t, "13800138000", "user@example.com")
			defer warden.Close()
			var calls atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "/v1/otp/challenges", r.URL.Path)
				require.Equal(t, "v3", r.Header.Get("X-Signature-Version"))
				require.Equal(t, "gateway", r.Header.Get("X-Key-Id"))
				require.NotEmpty(t, r.Header.Get("X-Signature"))
				require.Empty(t, r.Header.Get("X-API-Key"))
				require.Equal(t, "same-operation", r.Header.Get("Idempotency-Key"))
				w.Header().Set("Content-Type", "application/json")
				if tc.header != "" {
					w.Header().Set("Retry-After", tc.header)
				}
				w.WriteHeader(429)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer upstream.Close()
			t.Setenv("WARDEN_URL", warden.URL)
			t.Setenv("HERALD_URL", upstream.URL)
			auth.ResetWardenClientForTesting()
			ResetHeraldClientForTest()
			t.Cleanup(func() { auth.ResetWardenClientForTesting(); ResetHeraldClientForTest() })
			log := testLoggerSendVerifyCode()
			require.NoError(t, config.Initialize(log))
			require.NoError(t, auth.InitWardenClient(log))
			require.NoError(t, InitHeraldClient(log))
			app := fiber.New()
			app.Post("/_send_verify_code", SendVerifyCodeAPI())
			req := httptest.NewRequest("POST", "/_send_verify_code", strings.NewReader("phone=13800138000"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Idempotency-Key", "same-operation")
			response, err := app.Test(req)
			require.NoError(t, err)
			defer func() { _ = response.Body.Close() }()
			require.Equal(t, 429, response.StatusCode)
			var body map[string]any
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, "rate_limited", body["reason"])
			require.Equal(t, false, body["success"])
			if tc.want > 0 {
				require.Equal(t, float64(tc.want), body["next_resend_in"])
				require.Equal(t, fmt.Sprint(tc.want), response.Header.Get("Retry-After"))
			} else {
				require.NotContains(t, body, "next_resend_in")
				require.Empty(t, response.Header.Get("Retry-After"))
			}
			require.Equal(t, int32(1), calls.Load())
		})
	}
}

func TestChallengeErrorMetadataBoundsAndWrappedErrors(t *testing.T) {
	for _, n := range []int{-1, 0, 1, 86400, 86401} {
		ctx, app := createTestContext("POST", "/_send_verify_code", nil, "")
		require.NoError(t, handleChallengeCreationErrorWithMetadata(ctx, fmt.Errorf("wrapped: %w", &herald.HeraldError{StatusCode: 429}), herald.CreateChallengeMetadata{RetryAfterSeconds: &n}, "user", "email", "user@example.com"))
		require.Equal(t, 429, ctx.Response().StatusCode())
		require.Equal(t, n >= 1 && n <= 86400, len(ctx.Response().Header.Peek("Retry-After")) > 0)
		app.ReleaseCtx(ctx)
	}
	// Metadata must never turn a transport failure into a cooldown response.
	n := 30
	ctx, app := createTestContext("POST", "/_send_verify_code", nil, "")
	defer app.ReleaseCtx(ctx)
	require.NoError(t, handleChallengeCreationErrorWithMetadata(ctx, context.DeadlineExceeded, herald.CreateChallengeMetadata{RetryAfterSeconds: &n}, "user", "email", "user@example.com"))
	require.Equal(t, 500, ctx.Response().StatusCode())
	require.Empty(t, ctx.Response().Header.Peek("Retry-After"))
}
