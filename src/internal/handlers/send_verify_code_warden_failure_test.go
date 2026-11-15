package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/stretchr/testify/require"
)

// When the Warden lookup itself fails, POST /_send_verify_code must not tell
// the client, or the operator reading the log, that the user is unlisted:
// Warden refused Stargate's own credentials (401) or address or key (403),
// answered with an error, or did not answer. A real allow-list miss keeps its
// answer.
func TestSendVerifyCodeAPIDoesNotReportAFailedWardenLookupAsUnlistedUser(t *testing.T) {
	const (
		failedLookupLog = "Warden lookup failed; no verification code was sent"
		unlistedLog     = "User not found in Warden or not active"
	)
	tests := []struct {
		name        string
		status      int // 0: Warden does not answer
		wantLogs    []string
		wantStatus  int
		wantReason  string
		wantMessage string
	}{
		{
			name:        "warden refuses stargate credentials",
			status:      http.StatusUnauthorized,
			wantLogs:    []string{"Warden rejected this Stargate instance's own credentials", failedLookupLog, `"failure":"credentials_rejected"`},
			wantStatus:  fiber.StatusServiceUnavailable,
			wantReason:  "service_unavailable",
			wantMessage: "error.herald_unavailable_retry",
		},
		{
			name:        "warden refuses stargate address or key",
			status:      http.StatusForbidden,
			wantLogs:    []string{"Warden refused this Stargate instance with HTTP 403", failedLookupLog, `"failure":"access_denied"`},
			wantStatus:  fiber.StatusServiceUnavailable,
			wantReason:  "service_unavailable",
			wantMessage: "error.herald_unavailable_retry",
		},
		{
			name:        "warden server error",
			status:      http.StatusInternalServerError,
			wantLogs:    []string{failedLookupLog, `"failure":"server_error"`},
			wantStatus:  fiber.StatusServiceUnavailable,
			wantReason:  "service_unavailable",
			wantMessage: "error.herald_unavailable_retry",
		},
		{
			name:        "warden unavailable",
			status:      http.StatusServiceUnavailable,
			wantLogs:    []string{failedLookupLog, `"failure":"server_error"`},
			wantStatus:  fiber.StatusServiceUnavailable,
			wantReason:  "service_unavailable",
			wantMessage: "error.herald_unavailable_retry",
		},
		{
			name:        "warden rate limit",
			status:      http.StatusTooManyRequests,
			wantLogs:    []string{failedLookupLog, `"failure":"rate_limited"`},
			wantStatus:  fiber.StatusServiceUnavailable,
			wantReason:  "service_unavailable",
			wantMessage: "error.herald_unavailable_retry",
		},
		{
			name:        "warden does not answer",
			wantLogs:    []string{failedLookupLog, `"failure":"no_response"`},
			wantStatus:  fiber.StatusServiceUnavailable,
			wantReason:  "service_unavailable",
			wantMessage: "error.herald_unavailable_retry",
		},
		{
			name:        "user is not listed",
			status:      http.StatusNotFound,
			wantLogs:    []string{unlistedLog},
			wantStatus:  fiber.StatusUnauthorized,
			wantReason:  "user_not_in_list",
			wantMessage: "error.user_not_in_list",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSendVerifyCodeBaseEnv(t)
			t.Setenv("HERALD_ENABLED", "true")
			t.Setenv("WARDEN_API_KEY", "stargate-api-key-value")

			wardenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			t.Setenv("WARDEN_URL", wardenServer.URL)
			if tt.status == 0 {
				wardenServer.Close()
			} else {
				defer wardenServer.Close()
			}

			var heraldRequests atomic.Int32
			heraldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				heraldRequests.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer heraldServer.Close()
			t.Setenv("HERALD_URL", heraldServer.URL)
			t.Setenv("HERALD_API_KEY", "synthetic-only")

			var output bytes.Buffer
			testLog := logger.New(logger.Config{Level: logger.InfoLevel, Format: logger.FormatJSON, Output: &output})
			auth.ResetWardenClientForTesting()
			t.Cleanup(auth.ResetWardenClientForTesting)
			resetHeraldClientForTesting()
			t.Cleanup(resetHeraldClientForTesting)
			require.NoError(t, config.Initialize(testLog))
			require.NoError(t, auth.InitWardenClient(testLog))
			require.NoError(t, InitHeraldClient(testLog))
			SetLogger(testLog)
			// Only the lines of the request are examined below.
			output.Reset()

			ctx, app := createTestContext("POST", "/_send_verify_code", map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Accept":       "application/json",
			}, "phone=13800138000&mail=user%40example.com")
			defer app.ReleaseCtx(ctx)

			require.NoError(t, SendVerifyCodeAPI()(ctx))
			require.Equal(t, tt.wantStatus, ctx.Response().StatusCode())

			var body struct {
				Success     bool   `json:"success"`
				Message     string `json:"message"`
				Reason      string `json:"reason"`
				ChallengeID string `json:"challenge_id"`
			}
			require.NoError(t, json.Unmarshal(ctx.Response().Body(), &body))
			require.False(t, body.Success)
			require.Empty(t, body.ChallengeID, "no challenge is created when the lookup fails")
			require.Zero(t, heraldRequests.Load(), "Herald is not called when the lookup fails")
			require.Equal(t, tt.wantReason, body.Reason)
			require.Equal(t, i18n.TStatic(tt.wantMessage), body.Message)

			logged := output.String()
			for _, want := range tt.wantLogs {
				require.Contains(t, logged, want)
			}
			require.NotContains(t, logged, "stargate-api-key-value")
			if tt.status != http.StatusNotFound {
				require.NotContains(t, logged, "User not found in Warden")
				// A failed lookup is not about the user: the lines carry no
				// identifier, masked or not.
				for _, fragment := range []string{"13800138000", "8000", "user@example.com", "example.com", `"phone"`, `"mail"`} {
					require.NotContains(t, logged, fragment)
				}
			} else {
				require.NotContains(t, logged, failedLookupLog)
			}
		})
	}
}
