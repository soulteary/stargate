package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func TestWardenLoginBindsCanonicalProviderIdentity(t *testing.T) {
	variables := []*config.EnvVariable{&config.WardenEnabled, &config.WardenURL, &config.HeraldEnabled, &config.HeraldTOTPEnabled, &config.HeraldURL}
	old := make([]string, len(variables))
	for i, v := range variables {
		old[i] = v.Value
	}
	previousHerald := heraldClient
	t.Cleanup(func() {
		auth.ResetWardenClientForTesting()
		heraldClient = previousHerald
		for i, v := range variables {
			v.Value = old[i]
		}
	})
	canonical := warden.AllowListUser{UserID: "user-1", Mail: "canonical@example.com", Phone: "13800138000", Status: "active", Name: "Test User", Role: "viewer", Scope: []string{"read"}}
	seedLoginChallengeBinding(t, "challenge-1", "user-1", "email", canonical.Mail)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/user" {
			_ = json.NewEncoder(w).Encode(canonical)
			return
		}
		_ = json.NewEncoder(w).Encode(herald.VerifyChallengeV2Response{OK: true, UserID: "user-1", ChallengeID: "challenge-1", Purpose: "login", Channel: "email", AMR: []string{"otp", "email"}})
	}))
	defer server.Close()
	config.WardenEnabled.Value = "true"
	config.WardenURL.Value = server.URL
	config.HeraldEnabled.Value = "true"
	config.HeraldURL.Value = server.URL
	config.HeraldTOTPEnabled.Value = "false"
	auth.ResetWardenClientForTesting()
	require.NoError(t, auth.InitWardenClient(testLogger()))
	var err error
	heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
	require.NoError(t, err)
	store := setupTestStore()
	ctx, app := createTestContext("POST", "/_login", map[string]string{"Host": "auth.example.com", "Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json"}, "auth_method=warden&mail=canonical@example.com&challenge_id=challenge-1&verify_code=123456")
	defer app.ReleaseCtx(ctx)
	require.NoError(t, LoginAPI(store)(ctx))
	require.Equal(t, 200, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), `"success":true`)
	cookie := responseCookieValue(ctx, auth.SessionCookieName)
	require.NotEmpty(t, cookie)
	next, nextApp := createTestContext("GET", "/_auth", nil, "")
	defer nextApp.ReleaseCtx(next)
	next.Request().Header.SetCookie(auth.SessionCookieName, cookie)
	sess, err := store.Get(next)
	require.NoError(t, err)
	defer sess.Release()
	require.Equal(t, canonical.UserID, sess.Get("user_id"))
	require.Equal(t, canonical.Mail, sess.Get("user_mail"))
	require.Equal(t, canonical.Role, sess.Get("user_role"))
	require.Equal(t, canonical.Scope, sess.Get("user_scope"))
	require.Equal(t, []string{"otp", "email"}, sess.Get("user_amr"))
}
