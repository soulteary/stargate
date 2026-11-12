package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func sensitiveTestSession(t *testing.T, store *session.Store, method, path, body string) (fiber.Ctx, *fiber.App) {
	t.Helper()
	seed, app := createTestContext(method, path, map[string]string{"Host": "auth.example.com", "Content-Type": "application/x-www-form-urlencoded"}, body)
	sess, err := store.Get(seed)
	require.NoError(t, err)
	sess.Set("auth_method", "warden")
	sess.Set("user_id", "u1")
	sess.Set("user_mail", "user@example.com")
	sess.Set("auth_refreshed_at", time.Now().Unix())
	require.NoError(t, auth.Authenticate(sess))
	request := rebindSessionContext(t, app, seed, sess)
	sess.Release()
	app.ReleaseCtx(seed)
	return request, app
}

func TestSensitiveRoutesRejectUnavailableDirectory(t *testing.T) {
	setupRefreshTest(t)
	config.AuthRefreshEnabled.Value = "false"
	originalHerald := heraldClient
	t.Cleanup(func() { heraldClient = originalHerald })
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) }))
	defer server.Close()
	var err error
	heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
	require.NoError(t, err)
	lookups := 0
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		lookups++
		return nil, time.Time{}, errors.New("source expired")
	}
	store := setupTestStore()
	cases := []struct {
		name, method, path, body string
		handler                  func(fiber.Ctx) error
	}{
		{"enroll page", "GET", "/totp/enroll", "", TOTPEnrollPageRoute(store)},
		{"enroll start", "POST", "/totp/enroll", "", TOTPEnrollRoute(store)},
		{"enroll confirm", "POST", "/totp/enroll/confirm", "enroll_id=e1&code=123456", TOTPEnrollConfirmAPI(store)},
		{"revoke page", "GET", "/totp/revoke", "", TOTPRevokeRoute(store)},
		{"revoke factor", "POST", "/totp/revoke", "password=test123", TOTPRevokeConfirmAPI(store)},
		{"step-up page", "GET", "/_step_up", "", StepUpRoute(store)},
		{"step-up confirmation", "POST", "/_step_up", "password=test123", StepUpAPI(store)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, app := sensitiveTestSession(t, store, tc.method, tc.path, tc.body)
			defer app.ReleaseCtx(ctx)
			require.NoError(t, tc.handler(ctx))
			require.Equal(t, 503, ctx.Response().StatusCode())
			saved, err := store.Get(ctx)
			require.NoError(t, err)
			defer saved.Release()
			require.True(t, auth.IsAuthenticated(saved))
		})
	}
	require.Equal(t, len(cases), lookups)
	require.Zero(t, calls, "no factor-service side effect may precede directory authorization")
}

func TestSensitiveGuardDistinguishesRevocationAndRefresh(t *testing.T) {
	setupRefreshTest(t)
	store := setupTestStore()
	ctx, app := sensitiveTestSession(t, store, "POST", "/totp/enroll", "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return nil, time.Time{}, auth.ErrUserUnavailable
	}
	allowed, err := authorizeSensitiveSession(ctx, store, sess)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, 401, ctx.Response().StatusCode())
	require.False(t, auth.IsAuthenticated(sess))
	next, nextApp := sensitiveTestSession(t, store, "GET", "/totp/enroll", "")
	defer nextApp.ReleaseCtx(next)
	active, err := store.Get(next)
	require.NoError(t, err)
	defer active.Release()
	deadline := time.Now().Add(time.Minute)
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return &warden.AllowListUser{UserID: "u1", Mail: "user@example.com", Status: "active", Role: "viewer"}, deadline, nil
	}
	allowed, err = authorizeSensitiveSession(next, store, active)
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, "viewer", active.Get("user_role"))
	require.Equal(t, deadline.UnixNano(), active.Get("warden_source_valid_until"))
}

func TestSensitiveGuardRejectsAnonymousAndExpiredEvidence(t *testing.T) {
	setupRefreshTest(t)
	oldMax := config.WardenSourceMaxAge.Value
	t.Cleanup(func() { config.WardenSourceMaxAge.Value = oldMax })
	config.WardenSourceMaxAge.Value = "90s"
	store := setupTestStore()
	ctx, app := createTestContext("POST", "/totp/enroll", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	allowed, err := authorizeSensitiveSession(ctx, store, sess)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, 401, ctx.Response().StatusCode())
	next, nextApp := sensitiveTestSession(t, store, "POST", "/totp/enroll", "")
	defer nextApp.ReleaseCtx(next)
	active, err := store.Get(next)
	require.NoError(t, err)
	defer active.Release()
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return &warden.AllowListUser{UserID: "u1", Mail: "user@example.com", Status: "active"}, time.Now().Add(-time.Second), nil
	}
	allowed, err = authorizeSensitiveSession(next, store, active)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, 503, next.Response().StatusCode())
}
