package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/stretchr/testify/require"
)

// When Warden refuses Stargate's own credentials, POST /_send_verify_code must
// not tell the client, or the operator reading the log, that the user is
// unlisted. A real allow-list miss keeps its answer.
func TestSendVerifyCodeAPIDoesNotReportRejectedWardenCredentialsAsUnlistedUser(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		wantLog     string
		wantStatus  int
		wantReason  string
		wantMessage string
	}{
		{
			name:        "warden refuses stargate credentials",
			status:      http.StatusUnauthorized,
			wantLog:     "Warden rejected this Stargate instance's own credentials",
			wantStatus:  fiber.StatusServiceUnavailable,
			wantReason:  "service_unavailable",
			wantMessage: "error.herald_unavailable_retry",
		},
		{
			name:        "user is not listed",
			status:      http.StatusNotFound,
			wantLog:     "User not found in Warden or not active",
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
			defer wardenServer.Close()
			t.Setenv("WARDEN_URL", wardenServer.URL)

			var output bytes.Buffer
			testLog := logger.New(logger.Config{Level: logger.InfoLevel, Format: logger.FormatJSON, Output: &output})
			auth.ResetWardenClientForTesting()
			t.Cleanup(auth.ResetWardenClientForTesting)
			resetHeraldClientForTesting()
			require.NoError(t, config.Initialize(testLog))
			require.NoError(t, auth.InitWardenClient(testLog))
			SetLogger(testLog)

			ctx, app := createTestContext("POST", "/_send_verify_code", map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"Accept":       "application/json",
			}, "phone=13800138000")
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
			require.Equal(t, tt.wantReason, body.Reason)
			require.Equal(t, i18n.TStatic(tt.wantMessage), body.Message)

			logged := output.String()
			require.Contains(t, logged, tt.wantLog)
			require.NotContains(t, logged, "stargate-api-key-value")
			if tt.status != http.StatusNotFound {
				require.NotContains(t, logged, "User not found in Warden")
			}
		})
	}
}
