package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/redis/go-redis/v9"
	forwardauth "github.com/soulteary/forwardauth-kit/v3"
	"github.com/soulteary/herald/pkg/herald"
	sessionkit "github.com/soulteary/session-kit/v3"
	sessionfiber "github.com/soulteary/session-kit/v3/fiberadapter"
	"github.com/soulteary/session-kit/v3/redisstore"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func TestCanonicalEvidenceEmail(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Alice+tag@EXAMPLE.COM", "Alice+tag@example.com"},
		{"alice@bücher.example", "alice@xn--bcher-kva.example"},
		{" alice@example.com", ""}, {"Alice <alice@example.com>", ""},
		{"alice@example.com\r\nX-Foo: bar", ""}, {"alice@example.com.", ""},
		{"\"alice\"@example.com", ""}, {"älice@example.com", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, valid := canonicalEvidenceEmail(tc.input)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.want != "", valid)
		})
	}
}

func TestVerifiedEmailRequiresExactServerChallengeBinding(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mutate      func(*loginIdentity, *challengeContext)
		wantAddress string
	}{
		{"email", func(*loginIdentity, *challengeContext) {}, "Alice+tag@example.com"},
		{"sms", func(_ *loginIdentity, e *challengeContext) { e.Channel = "sms" }, ""},
		{"wrong challenge", func(_ *loginIdentity, e *challengeContext) { e.ChallengeID = "other" }, ""},
		{"wrong subject", func(_ *loginIdentity, e *challengeContext) { e.UserID = "other" }, ""},
		{"wrong destination", func(_ *loginIdentity, e *challengeContext) { e.Destination = "different@example.com" }, ""},
		{"local case differs", func(_ *loginIdentity, e *challengeContext) { e.Destination = "alice+tag@example.com" }, ""},
		{"plus tag differs", func(_ *loginIdentity, e *challengeContext) { e.Destination = "Alice@example.com" }, ""},
		{"missing context", func(_ *loginIdentity, e *challengeContext) { *e = challengeContext{} }, ""},
		{"no explicit subject", func(i *loginIdentity, _ *challengeContext) { i.wardenUserInfo.UserID = "" }, ""},
		{"no provider", func(i *loginIdentity, _ *challengeContext) { i.wardenUserInfo = nil }, ""},
		{"provider subject changed", func(i *loginIdentity, _ *challengeContext) { i.wardenUserInfo.UserID = "other" }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			identity := &loginIdentity{userID: "user-1", verifyRespAMR: []string{"otp", "email"}, wardenUserInfo: &warden.AllowListUser{UserID: "user-1", Mail: "Alice+tag@EXAMPLE.COM"}}
			evidence := challengeContext{ChallengeID: "challenge-1", UserID: "user-1", Channel: "email", Destination: "Alice+tag@example.com"}
			tc.mutate(identity, &evidence)
			identity.recordVerifiedEmail("challenge-1", evidence, time.Unix(100, 0))
			require.Equal(t, tc.wantAddress, identity.verifiedEmail)
		})
	}
}

func seedVerifiedEmail(t *testing.T, sess *session.Session, now time.Time) {
	t.Helper()
	identity := &loginIdentity{userID: "user-1", verifiedEmail: "alice@example.com", verifiedAt: now.Unix(), wardenUserInfo: &warden.AllowListUser{UserID: "user-1", Mail: "alice@example.com"}}
	sess.Set("auth_method", "warden")
	storeLoginIdentity(sess, "warden", identity)
}

func TestVerifiedEmailHeaderRejectsInvalidOrStaleSessionEvidence(t *testing.T) {
	now := time.Unix(1800000000, 0)
	for _, tc := range []struct {
		name   string
		mutate func(*session.Session, *forwardauth.AuthResult)
		want   string
	}{
		{"valid", func(*session.Session, *forwardauth.AuthResult) {}, "true"},
		{"legacy AMR only", func(s *session.Session, _ *forwardauth.AuthResult) {
			clearVerifiedEmail(s)
			s.Set("user_amr", []string{"otp", "email"})
		}, "false"},
		{"password result", func(_ *session.Session, r *forwardauth.AuthResult) { r.AuthMethod = forwardauth.AuthMethodPassword }, "false"},
		{"trusted header result", func(_ *session.Session, r *forwardauth.AuthResult) { r.AuthMethod = forwardauth.AuthMethodHeader }, "false"},
		{"unauthenticated", func(_ *session.Session, r *forwardauth.AuthResult) { r.Authenticated = false }, "false"},
		{"wrong subject", func(_ *session.Session, r *forwardauth.AuthResult) { r.UserID = "other" }, "false"},
		{"wrong email", func(_ *session.Session, r *forwardauth.AuthResult) { r.Email = "other@example.com" }, "false"},
		{"domain case", func(_ *session.Session, r *forwardauth.AuthResult) { r.Email = "alice@EXAMPLE.COM" }, "true"},
		{"wrong source", func(s *session.Session, _ *forwardauth.AuthResult) { s.Set("verified_email_source", "sms") }, "false"},
		{"wrong version", func(s *session.Session, _ *forwardauth.AuthResult) { s.Set("verified_email_version", int64(2)) }, "false"},
		{"wrong timestamp type", func(s *session.Session, _ *forwardauth.AuthResult) { s.Set("verified_email_at", "1800000000") }, "false"},
		{"future", func(s *session.Session, _ *forwardauth.AuthResult) {
			s.Set("verified_email_at", now.Add(time.Minute).Unix())
		}, "false"},
		{"expired", func(s *session.Session, _ *forwardauth.AuthResult) {
			s.Set("verified_email_at", now.Add(-config.SessionExpiration).Unix())
		}, "false"},
		{"password session", func(s *session.Session, _ *forwardauth.AuthResult) { s.Set("auth_method", "password") }, "false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, app := createTestContext("GET", "/_auth", nil, "")
			defer app.ReleaseCtx(ctx)
			sess, err := setupTestStore().Get(ctx)
			require.NoError(t, err)
			defer sess.Release()
			seedVerifiedEmail(t, sess, now)
			result := &forwardauth.AuthResult{Authenticated: true, AuthMethod: forwardauth.AuthMethodSession, UserID: "user-1", Email: "alice@example.com"}
			tc.mutate(sess, result)
			setVerifiedEmailHeader(ctx, sess, result, now)
			require.Equal(t, tc.want, string(ctx.Response().Header.Peek(verifiedEmailHeader)))
		})
	}
}

func TestVerifiedEmailIsClearedByIdentityReplacementAndDirectoryChange(t *testing.T) {
	setupRefreshTest(t)
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := setupTestStore().Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	seedVerifiedEmail(t, sess, time.Now())
	require.NoError(t, auth.Authenticate(sess))
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return &warden.AllowListUser{UserID: "user-1", Mail: "changed@example.com", Status: "active"}, time.Time{}, nil
	}
	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	require.NoError(t, err)
	require.True(t, refreshed)
	for _, key := range verifiedEmailKeys {
		require.Nil(t, sess.Get(key))
	}
	// A later refresh restoring the old email must not restore the old proof.
	sess.Delete("auth_refreshed_at")
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return &warden.AllowListUser{UserID: "user-1", Mail: "alice@example.com", Status: "active"}, time.Time{}, nil
	}
	_, err = refreshAuthorizationIfNeeded(context.Background(), sess)
	require.NoError(t, err)
	require.Nil(t, sess.Get("verified_email_at"))
	seedVerifiedEmail(t, sess, time.Now())
	storeLoginIdentity(sess, "password", &loginIdentity{})
	for _, key := range verifiedEmailKeys {
		require.Nil(t, sess.Get(key))
	}
}

func TestVerifiedEmailLoginToForwardAuth(t *testing.T) {
	for _, tc := range []struct {
		name, channel, destination, evidenceSubject, responseSubject string
		want                                                         string
		status                                                       int
	}{
		{"email", "email", "alice@EXAMPLE.COM", "user-1", "user-1", "true", 200},
		{"sms AMR email is not proof", "sms", "13800138000", "user-1", "user-1", "false", 200},
		{"different email", "email", "other@example.com", "user-1", "user-1", "false", 200},
		{"different context subject", "email", "alice@example.com", "other", "user-1", "false", 200},
		{"missing context", "", "", "", "user-1", "false", 200},
		{"Herald subject mismatch", "email", "alice@example.com", "user-1", "other", "", 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupCheckHeaderConfig(t)
			variables := []*config.EnvVariable{&config.WardenEnabled, &config.WardenURL, &config.HeraldEnabled, &config.HeraldTOTPEnabled, &config.HeraldURL, &config.AuthRefreshEnabled}
			old := make([]string, len(variables))
			for i, v := range variables {
				old[i] = v.Value
			}
			previousHerald, previousContexts := heraldClient, getChallengeContextStore()
			t.Cleanup(func() {
				auth.ResetWardenClientForTesting()
				heraldClient = previousHerald
				SetChallengeContextStore(previousContexts)
				for i, v := range variables {
					v.Value = old[i]
				}
			})
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/user" {
					_ = json.NewEncoder(w).Encode(warden.AllowListUser{UserID: "user-1", Mail: "alice@example.com", Status: "active"})
					return
				}
				_ = json.NewEncoder(w).Encode(herald.VerifyChallengeResponse{OK: true, UserID: tc.responseSubject, AMR: []string{"otp", "email"}})
			}))
			defer provider.Close()
			config.WardenEnabled.Value, config.WardenURL.Value = "true", provider.URL
			config.HeraldEnabled.Value, config.HeraldURL.Value = "true", provider.URL
			config.HeraldTOTPEnabled.Value, config.AuthRefreshEnabled.Value = "false", "false"
			auth.ResetWardenClientForTesting()
			require.NoError(t, auth.InitWardenClient(testLogger()))
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(provider.URL))
			require.NoError(t, err)
			contexts := newMemoryChallengeContextStore()
			SetChallengeContextStore(contexts)
			if tc.channel != "" {
				_, err := contexts.PutIfAbsent(context.Background(), challengeContext{ChallengeID: "challenge-1", UserID: tc.evidenceSubject, Channel: tc.channel, Destination: tc.destination}, time.Minute)
				require.NoError(t, err)
			}
			store := setupTestStore()
			form := url.Values{"auth_method": {"warden"}, "mail": {"alice@example.com"}, "challenge_id": {"challenge-1"}, "verify_code": {"123456"}}
			ctx, app := createTestContext("POST", "/_login", map[string]string{"Host": "auth.example.com", "Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json"}, form.Encode())
			defer app.ReleaseCtx(ctx)
			require.NoError(t, LoginAPI(store)(ctx))
			require.Equal(t, tc.status, ctx.Response().StatusCode())
			if tc.status != 200 {
				return
			}
			cookie := responseCookieValue(ctx, auth.SessionCookieName)
			require.NotEmpty(t, cookie)
			next, nextApp := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json", verifiedEmailHeader: "true"}, "")
			defer nextApp.ReleaseCtx(next)
			next.Request().Header.SetCookie(auth.SessionCookieName, cookie)
			require.NoError(t, CheckRoute(store)(next))
			require.Equal(t, 200, next.Response().StatusCode())
			require.Equal(t, tc.want, string(next.Response().Header.Peek(verifiedEmailHeader)))
			require.Empty(t, next.Get(verifiedEmailHeader))
		})
	}
}

func TestVerifiedEmailFailureDoesNotExportHeader(t *testing.T) {
	setupCheckHeaderConfig(t)
	ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json", verifiedEmailHeader: "true"}, "")
	defer app.ReleaseCtx(ctx)
	ctx.Set(verifiedEmailHeader, "true") // Guard against stale middleware output.
	require.NoError(t, CheckRoute(setupTestStore())(ctx))
	require.Equal(t, 401, ctx.Response().StatusCode())
	require.Empty(t, ctx.Response().Header.Peek(verifiedEmailHeader))
	require.Empty(t, ctx.Get(verifiedEmailHeader))
}

func TestVerifiedEmailRedisSessionRoundTrip(t *testing.T) {
	setupCheckHeaderConfig(t)
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	storage := redisstore.New(client, "verified-email-test:")
	manager := sessionkit.NewManager(storage, sessionkit.DefaultConfig().WithCookieName(auth.SessionCookieName))
	store := session.NewStore(sessionfiber.SessionConfig(manager))
	app := fiber.New()
	app.Get("/seed", func(c fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return err
		}
		defer sess.Release()
		seedVerifiedEmail(t, sess, time.Now())
		return auth.Authenticate(sess)
	})
	app.Get("/_auth", CheckRoute(store))
	login, err := app.Test(httptest.NewRequest("GET", "http://localhost/seed", nil))
	require.NoError(t, err)
	defer func() { require.NoError(t, login.Body.Close()) }()
	require.NotEmpty(t, login.Cookies())
	request := httptest.NewRequest("GET", "http://localhost/_auth", nil)
	request.Header.Set("Accept", "application/json")
	for _, cookie := range login.Cookies() {
		request.AddCookie(cookie)
	}
	response, err := app.Test(request)
	require.NoError(t, err)
	defer func() { require.NoError(t, response.Body.Close()) }()
	require.Equal(t, 200, response.StatusCode)
	require.Equal(t, "true", response.Header.Get(verifiedEmailHeader))
}
