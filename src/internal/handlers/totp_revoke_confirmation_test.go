package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

// Herald can answer the revoke call with 200 and ok:false. The user must not be
// told the factor was removed in that case.
func TestTOTPRevokeRequiresHeraldConfirmation(t *testing.T) {
	originalHerald := heraldClient
	t.Cleanup(func() { heraldClient = originalHerald })
	setConfigValue(t, &config.HeraldTOTPEnabled, "true")

	for _, tc := range []struct {
		name       string
		confirmed  bool
		wantStatus int
		wantBody   string
	}{
		{name: "confirmed", confirmed: true, wantStatus: fiber.StatusOK, wantBody: `"ok":true`},
		{name: "not confirmed", confirmed: false, wantStatus: fiber.StatusBadGateway, wantBody: "revoke_not_confirmed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			revokeCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/status") {
					_ = json.NewEncoder(w).Encode(map[string]any{"totp_enabled": true})
					return
				}
				if strings.HasSuffix(r.URL.Path, "/verify") {
					_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
					return
				}
				revokeCalls++
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": tc.confirmed, "subject": "u1"})
			}))
			defer server.Close()

			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)

			store := setupTestStore()
			seed, app := createTestContext(fiber.MethodPost, "/totp/revoke", map[string]string{
				"Host":         "auth.example.com",
				"Content-Type": "application/x-www-form-urlencoded",
			}, "code=123456")
			sess, err := store.Get(seed)
			require.NoError(t, err)
			sess.Set("auth_method", "password")
			sess.Set("user_id", "u1")
			require.NoError(t, auth.Authenticate(sess))
			ctx := rebindSessionContext(t, app, seed, sess)
			sess.Release()
			app.ReleaseCtx(seed)
			defer app.ReleaseCtx(ctx)

			require.NoError(t, TOTPRevokeConfirmAPI(store)(ctx))
			require.Equal(t, tc.wantStatus, ctx.Response().StatusCode(), string(ctx.Response().Body()))
			require.Contains(t, string(ctx.Response().Body()), tc.wantBody)
			require.Equal(t, 1, revokeCalls)
		})
	}
}
