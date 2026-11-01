package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationRefreshCannotRebindReassignedContact(t *testing.T) {
	setupRefreshTest(t)
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := setupTestStore().Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	sess.Set("auth_method", "warden")
	sess.Set("user_id", "original-user")
	sess.Set("user_mail", "reassigned@example.com")
	sess.Set("user_role", "viewer")
	require.NoError(t, auth.Authenticate(sess))
	lookupRefreshUser = func(context.Context, string, string) *warden.AllowListUser {
		return &warden.AllowListUser{UserID: "new-owner", Mail: "reassigned@example.com", Status: "active", Role: "admin"}
	}
	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	require.Error(t, err)
	require.False(t, refreshed)
	require.Equal(t, "original-user", sess.Get("user_id"))
	require.Equal(t, "viewer", sess.Get("user_role"))
}

func TestAuthorizationRefreshRequiresStableIdentityForLegacySessions(t *testing.T) {
	for _, tc := range []struct {
		name, bound, provider string
		allow                 bool
	}{
		{"same identity", "user-1", "user-1", true},
		{"legacy missing identity", "", "user-1", false},
		{"directory missing identity", "user-1", "", false},
		{"both missing identity", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupRefreshTest(t)
			ctx, app := createTestContext("GET", "/_auth", nil, "")
			defer app.ReleaseCtx(ctx)
			sess, err := setupTestStore().Get(ctx)
			require.NoError(t, err)
			defer sess.Release()
			sess.Set("user_id", tc.bound)
			sess.Set("user_mail", "user@example.com")
			// Missing auth_method models a legacy Warden session.
			require.NoError(t, auth.Authenticate(sess))
			lookupRefreshUser = func(context.Context, string, string) *warden.AllowListUser {
				return &warden.AllowListUser{UserID: tc.provider, Mail: "user@example.com", Status: "active", Role: "viewer"}
			}
			refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
			require.Equal(t, tc.allow, refreshed)
			if tc.allow {
				require.NoError(t, err)
				require.Equal(t, "viewer", sess.Get("user_role"))
			} else {
				require.Error(t, err)
				require.Equal(t, tc.bound, sess.Get("user_id"))
			}
		})
	}
}

func TestForwardAuthRejectsReassignedDirectoryIdentity(t *testing.T) {
	setupRefreshTest(t)
	previousURL := config.WardenURL.Value
	t.Cleanup(func() { config.WardenURL.Value = previousURL; auth.ResetWardenClientForTesting() })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(warden.AllowListUser{UserID: "new-owner", Mail: "reassigned@example.com", Status: "active", Role: "admin"})
	}))
	defer server.Close()
	config.WardenURL.Value = server.URL
	auth.ResetWardenClientForTesting()
	require.NoError(t, auth.InitWardenClient(testLogger()))
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json"}, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	sess.Set("user_id", "original-user")
	sess.Set("user_mail", "reassigned@example.com")
	sess.Set("auth_method", "warden")
	require.NoError(t, auth.Authenticate(sess))
	request := rebindSessionContext(t, app, ctx, sess)
	defer app.ReleaseCtx(request)
	sess.Release()
	require.NoError(t, CheckRoute(store)(request))
	require.Equal(t, 401, request.Response().StatusCode())
	require.Empty(t, request.GetRespHeader(config.UserHeaderName.Value))
}
