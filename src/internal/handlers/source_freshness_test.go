package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func TestSourceDeadlineRefreshesBeforeSessionInterval(t *testing.T) {
	setupRefreshTest(t)
	previous := config.WardenSourceMaxAge.Value
	t.Cleanup(func() { config.WardenSourceMaxAge.Value = previous })
	config.WardenSourceMaxAge.Value = "90s"
	config.AuthRefreshInterval.Value = "5m"
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	sess.Set("auth_method", "warden")
	sess.Set("user_mail", "active@example.com")
	sess.Set("user_id", "u1")
	sess.Set("auth_refreshed_at", time.Now().Unix())
	require.NoError(t, auth.Authenticate(sess))
	require.True(t, wardenSessionNeedsRefresh(sess), "sessions from before strict mode must revalidate")
	sess.Set("warden_source_policy", wardenSourcePolicy())
	sess.Set("warden_source_valid_until", time.Now().Add(time.Minute).UnixNano())
	require.False(t, wardenSessionNeedsRefresh(sess))
	config.WardenSourceMaxAge.Value = "30s"
	require.True(t, wardenSessionNeedsRefresh(sess), "tightened policies must revalidate existing evidence")
	config.WardenSourceMaxAge.Value = "90s"
	sess.Set("warden_source_valid_until", time.Now().Add(-time.Second).UnixNano())
	require.True(t, wardenSessionNeedsRefresh(sess))
	deadline := time.Now().Add(time.Minute)
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return &warden.AllowListUser{UserID: "u1", Mail: "active@example.com", Status: "active"}, deadline, nil
	}
	updated, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	require.NoError(t, err)
	require.True(t, updated)
	require.Equal(t, deadline.UnixNano(), sess.Get("warden_source_valid_until"))
}

func TestDirectoryFailurePreservesSessionWithoutExtendingDeadline(t *testing.T) {
	setupRefreshTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	sess.Set("auth_method", "warden")
	sess.Set("user_mail", "active@example.com")
	sess.Set("user_id", "u1")
	oldDeadline := time.Now().Add(-time.Second).UnixNano()
	sess.Set("warden_source_valid_until", oldDeadline)
	require.NoError(t, auth.Authenticate(sess))
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return nil, time.Time{}, errors.New("source expired")
	}
	updated, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	require.ErrorIs(t, err, errAuthorizationUnavailable)
	require.False(t, updated)
	require.True(t, auth.IsAuthenticated(sess))
	require.Equal(t, oldDeadline, sess.Get("warden_source_valid_until"))
}

func TestMetricsRequiresSeparateBearerCredential(t *testing.T) {
	previous := config.MetricsAPIKey.Value
	t.Cleanup(func() { config.MetricsAPIKey.Value = previous })
	key := strings.Repeat("k", 32)
	for _, tt := range []struct {
		name, configured, header string
		status                   int
	}{
		{"disabled", "", "", 404}, {"missing", key, "", 401}, {"raw key", key, key, 401}, {"empty bearer", key, "Bearer ", 401}, {"wrong", key, "Bearer wrong", 401}, {"valid", key, "Bearer " + key, 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			config.MetricsAPIKey.Value = tt.configured
			app := fiber.New()
			app.Get("/metrics", MetricsAuth(), func(c fiber.Ctx) error { return c.SendString("metric 1") })
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			req.Header.Set("Authorization", tt.header)
			response, err := app.Test(req)
			require.NoError(t, err)
			require.Equal(t, tt.status, response.StatusCode)
			require.NoError(t, response.Body.Close())
		})
	}
}

func TestForwardAuthSourceFailureReturns503AndKeepsSession(t *testing.T) {
	setupRefreshTest(t)
	previous := config.WardenSourceMaxAge.Value
	t.Cleanup(func() { config.WardenSourceMaxAge.Value = previous })
	config.WardenSourceMaxAge.Value = "90s"
	InitForwardAuthHandler(testLogger())
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json"}, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	sess.Set("user_id", "u1")
	sess.Set("user_mail", "active@example.com")
	sess.Set("auth_method", "warden")
	sess.Set("auth_refreshed_at", time.Now().Unix())
	sess.Set("warden_source_valid_until", time.Now().Add(-time.Second).UnixNano())
	require.NoError(t, auth.Authenticate(sess))
	request := rebindSessionContext(t, app, ctx, sess)
	defer app.ReleaseCtx(request)
	sess.Release()
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return nil, time.Time{}, errors.New("source expired")
	}
	require.NoError(t, CheckRoute(store)(request))
	require.Equal(t, 503, request.Response().StatusCode())
	require.Empty(t, request.GetRespHeader("X-Auth-User"))
	saved, err := store.Get(request)
	require.NoError(t, err)
	defer saved.Release()
	require.True(t, auth.IsAuthenticated(saved))
}

func TestTrustedHeaderDependencyFailureHasNoAuthenticationResult(t *testing.T) {
	previous := lookupTrustedHeaderUser
	t.Cleanup(func() { lookupTrustedHeaderUser = previous })
	lookupTrustedHeaderUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return nil, time.Time{}, errors.New("source expired")
	}
	result, _, err := trustedHeaderAuthResult(context.Background(), "", "active@example.com")
	require.Nil(t, result)
	require.ErrorIs(t, err, errAuthorizationUnavailable)
}

func TestForwardAuthRechecksSourceDeadlineBeforeEmittingIdentity(t *testing.T) {
	setupRefreshTest(t)
	previousMax, previousNow := config.WardenSourceMaxAge.Value, authorizationNow
	t.Cleanup(func() { config.WardenSourceMaxAge.Value = previousMax; authorizationNow = previousNow })
	config.WardenSourceMaxAge.Value = "90s"
	InitForwardAuthHandler(testLogger())
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json"}, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	now := time.Now()
	deadline := now.Add(time.Minute)
	sess.Set("user_id", "u1")
	sess.Set("user_mail", "active@example.com")
	sess.Set("auth_method", "warden")
	sess.Set("auth_refreshed_at", now.Unix())
	sess.Set("warden_source_valid_until", deadline.UnixNano())
	sess.Set("warden_source_policy", wardenSourcePolicy())
	require.NoError(t, auth.Authenticate(sess))
	request := rebindSessionContext(t, app, ctx, sess)
	defer app.ReleaseCtx(request)
	sess.Release()
	calls := 0
	authorizationNow = func() time.Time {
		calls++
		if calls == 1 {
			return now
		}
		return deadline
	}
	require.NoError(t, CheckRoute(store)(request))
	require.Equal(t, 503, request.Response().StatusCode())
	require.Empty(t, request.GetRespHeader("X-Auth-User"))
	require.Equal(t, 2, calls)
}

func TestForwardAuthRejectsExpiredTrustedHeaderEvidence(t *testing.T) {
	previousMax, previousLookup := config.WardenSourceMaxAge.Value, lookupTrustedHeaderUser
	previousEnabled, previousSecret := config.HeaderAuthEnabled.Value, config.HeaderAuthSharedSecret.Value
	t.Cleanup(func() {
		config.WardenSourceMaxAge.Value = previousMax
		lookupTrustedHeaderUser = previousLookup
		config.HeaderAuthEnabled.Value = previousEnabled
		config.HeaderAuthSharedSecret.Value = previousSecret
	})
	config.WardenSourceMaxAge.Value = "90s"
	config.HeaderAuthEnabled.Value = "true"
	config.HeaderAuthSharedSecret.Value = strings.Repeat("s", 32)
	InitForwardAuthHandler(testLogger())
	for _, tt := range []struct {
		name     string
		deadline time.Time
		status   int
	}{{"expired", time.Now().Add(-time.Second), 503}, {"missing", time.Time{}, 503}, {"valid", time.Now().Add(time.Minute), 200}} {
		t.Run(tt.name, func(t *testing.T) {
			lookupTrustedHeaderUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
				return &warden.AllowListUser{UserID: "u1", Mail: "active@example.com", Status: "active"}, tt.deadline, nil
			}
			ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json", "X-User-Mail": "active@example.com", config.HeaderAuthSecretHeader.Value: config.HeaderAuthSharedSecret.Value}, "")
			defer app.ReleaseCtx(ctx)
			require.NoError(t, CheckRoute(setupTestStore())(ctx))
			require.Equal(t, tt.status, ctx.Response().StatusCode())
			if tt.status == 503 {
				require.Empty(t, ctx.GetRespHeader("X-Auth-User"))
			}
		})
	}
}

func TestWardenLoginMissingIdentityReturns401(t *testing.T) {
	setupRefreshTest(t)
	oldURL := config.WardenURL.Value
	t.Cleanup(func() { config.WardenURL.Value = oldURL; auth.ResetWardenClientForTesting() })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	defer server.Close()
	config.WardenURL.Value = server.URL
	auth.ResetWardenClientForTesting()
	require.NoError(t, auth.InitWardenClient(testLogger()))
	ctx, app := createTestContext("POST", "/_login", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "auth_method=warden&mail=missing@example.com")
	defer app.ReleaseCtx(ctx)
	require.NoError(t, LoginAPI(setupTestStore())(ctx))
	require.Equal(t, 401, ctx.Response().StatusCode())
}

func TestWardenLoginChecksSourceDeadlineAfterOTP(t *testing.T) {
	variables := []*config.EnvVariable{&config.WardenEnabled, &config.WardenURL, &config.WardenSourceMaxAge, &config.WardenSourceClockSkew, &config.HeraldEnabled, &config.HeraldTOTPEnabled, &config.HeraldURL}
	old := make([]string, len(variables))
	for i, v := range variables {
		old[i] = v.Value
	}
	oldHerald, oldNow := heraldClient, authorizationNow
	t.Cleanup(func() {
		auth.ResetWardenClientForTesting()
		heraldClient = oldHerald
		authorizationNow = oldNow
		for i, v := range variables {
			v.Value = old[i]
		}
	})
	now := time.Now().Round(0)
	deadline := now.Add(time.Minute)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/user" {
			w.Header().Set("Warden-Snapshot-Metadata", "v1")
			w.Header().Set("Warden-Snapshot-Version", strings.Repeat("a", 64))
			w.Header().Set("Warden-Source-Succeeded-At", now.Add(-10*time.Second).Format(time.RFC3339Nano))
			w.Header().Set("Warden-Snapshot-Valid-Until", deadline.Format(time.RFC3339Nano))
			_, _ = w.Write([]byte(`{"user_id":"u1","mail":"active@example.com","status":"active"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"user_id":"u1","challenge_id":"challenge-1","purpose":"login","channel":"email","amr":["otp","email"]}`))
	}))
	defer server.Close()
	config.WardenEnabled.Value = "true"
	config.WardenURL.Value = server.URL
	config.WardenSourceMaxAge.Value = "90s"
	config.WardenSourceClockSkew.Value = "0s"
	config.HeraldEnabled.Value = "true"
	config.HeraldTOTPEnabled.Value = "false"
	config.HeraldURL.Value = server.URL
	auth.ResetWardenClientForTesting()
	require.NoError(t, auth.InitWardenClient(testLogger()))
	var err error
	heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
	require.NoError(t, err)
	for _, tc := range []struct {
		name     string
		authTime time.Time
		status   int
	}{{"expired after OTP", deadline, 503}, {"valid source", now, 200}} {
		t.Run(tc.name, func(t *testing.T) {
			seedLoginChallengeBinding(t, "challenge-1", "u1", "email", "active@example.com")
			authorizationNow = func() time.Time { return tc.authTime }
			store := setupTestStore()
			ctx, app := createTestContext("POST", "/_login", map[string]string{"Host": "auth.example.com", "Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json"}, "auth_method=warden&mail=active@example.com&challenge_id=challenge-1&verify_code=123456")
			defer app.ReleaseCtx(ctx)
			require.NoError(t, LoginAPI(store)(ctx))
			require.Equal(t, tc.status, ctx.Response().StatusCode())
			if tc.status == 503 {
				require.Empty(t, responseCookieValue(ctx, auth.SessionCookieName))
				return
			}
			cookie := responseCookieValue(ctx, auth.SessionCookieName)
			next, nextApp := createTestContext("GET", "/_auth", nil, "")
			defer nextApp.ReleaseCtx(next)
			next.Request().Header.SetCookie(auth.SessionCookieName, cookie)
			sess, err := store.Get(next)
			require.NoError(t, err)
			defer sess.Release()
			require.Equal(t, deadline.UnixNano(), sess.Get("warden_source_valid_until"))
			require.Equal(t, wardenSourcePolicy(), sess.Get("warden_source_policy"))
		})
	}
}
