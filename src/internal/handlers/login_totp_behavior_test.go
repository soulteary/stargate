package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoginTOTPRequiresEnrolledIdentityAndSuccessfulVerification(t *testing.T) {
	oldClient, oldEnabled := heraldClient, config.HeraldTOTPEnabled.Value
	t.Cleanup(func() { heraldClient = oldClient; config.HeraldTOTPEnabled.Value = oldEnabled })
	for _, tc := range []struct {
		name, code                   string
		enabled, enrolled, verified  bool
		statusFailure, verifyFailure bool
		want                         int
	}{
		{name: "missing code", enabled: true, want: 400},
		{name: "disabled", code: "123456", want: 503},
		{name: "not enrolled", code: "123456", enabled: true, want: 400},
		{name: "status unavailable", code: "123456", enabled: true, statusFailure: true, want: 502},
		{name: "invalid code", code: "123456", enabled: true, enrolled: true, want: 401},
		{name: "verification unavailable", code: "123456", enabled: true, enrolled: true, verifyFailure: true, want: 401},
		{name: "verified", code: "123456", enabled: true, enrolled: true, verified: true, want: 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var verifiedSubject string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/status") {
					if tc.statusFailure {
						w.WriteHeader(503)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"subject": "user-1", "totp_enabled": tc.enrolled})
					return
				}
				var request herald.TOTPVerifyRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				verifiedSubject = request.Subject
				if tc.verifyFailure {
					w.WriteHeader(503)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": tc.verified})
			}))
			defer server.Close()
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)
			config.HeraldTOTPEnabled.Value = "false"
			if tc.enabled {
				config.HeraldTOTPEnabled.Value = "true"
			}
			app := fiber.New()
			app.Post("/", func(ctx fiber.Ctx) error {
				identity := &loginIdentity{userID: "user-1"}
				ok, err := identity.verifyTOTP(ctx, context.Background())
				require.Equal(t, tc.verified, ok)
				if !ok {
					return err
				}
				return ctx.SendStatus(204)
			})
			request := httptest.NewRequest("POST", "/", strings.NewReader("otp_code="+tc.code+"&challenge_id=test-challenge"))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err := app.Test(request)
			require.NoError(t, err)
			require.Equal(t, tc.want, response.StatusCode)
			require.NoError(t, response.Body.Close())
			if tc.enrolled {
				require.Equal(t, "user-1", verifiedSubject)
			}
		})
	}
}
