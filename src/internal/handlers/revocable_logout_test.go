package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/redis/go-redis/v9"
	sessionkit "github.com/soulteary/session-kit/v3"
	sessionfiber "github.com/soulteary/session-kit/v3/fiberadapter"
	"github.com/soulteary/session-kit/v3/redisstore"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func revocableTestStore(t *testing.T, backend string) (*session.Store, func() int) {
	t.Helper()
	var storage sessionkit.Storage
	var count func() int
	if backend == "memory" {
		memory := sessionkit.NewMemoryStorage("test:revocable:", 0)
		storage = memory
		count = memory.Len
		t.Cleanup(func() { require.NoError(t, memory.Close()) })
	} else {
		server := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: server.Addr()})
		t.Cleanup(func() { require.NoError(t, client.Close()) })
		storage = redisstore.New(client, "test:revocable:")
		count = func() int { return len(server.Keys()) }
	}
	manager := sessionkit.NewManager(storage, sessionkit.DefaultConfig().WithCookieName(auth.SessionCookieName))
	cfg, err := sessionfiber.RevocableSessionConfig(manager)
	require.NoError(t, err)
	return session.NewStore(cfg), count
}

func TestAnonymousLogoutDoesNotAllocateRevocationMarkers(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			store, count := revocableTestStore(t, backend)
			app := fiber.New()
			app.Post("/_logout", RequireSameOrigin(), LogoutRoute(store))
			for _, cookie := range []string{"", "unknown-cookie", sessionkit.NewRevocableSessionID(time.Hour)} {
				for i := 0; i < 3; i++ {
					request := httptest.NewRequest("POST", "http://auth.example.com/_logout", nil)
					if cookie != "" {
						request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: cookie})
					}
					response, err := app.Test(request)
					require.NoError(t, err)
					require.Equal(t, 200, response.StatusCode)
					require.NoError(t, response.Body.Close())
					cleared := false
					for _, c := range response.Cookies() {
						if c.Name == auth.SessionCookieName && c.Value == "" {
							cleared = true
						}
					}
					require.True(t, cleared)
				}
			}
			require.Zero(t, count())
		})
	}
}

func TestLogoutRejectsAnInflightAuthenticatedRefresh(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			setupRefreshTest(t)
			InitForwardAuthHandler(testLogger())
			store, _ := revocableTestStore(t, backend)
			seed, seedApp := createTestContext("GET", "/_auth", nil, "")
			defer seedApp.ReleaseCtx(seed)
			sess, err := store.Get(seed)
			require.NoError(t, err)
			sess.Set("auth_method", "warden")
			sess.Set("user_id", "u1")
			sess.Set("user_mail", "user@example.com")
			sess.Set("auth_refreshed_at", time.Now().Add(-time.Hour).Unix())
			require.NoError(t, auth.Authenticate(sess))
			id := sess.ID()
			sess.Release()
			entered, resume := make(chan struct{}), make(chan struct{})
			lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
				close(entered)
				<-resume
				return &warden.AllowListUser{UserID: "u1", Mail: "user@example.com", Status: "active"}, time.Time{}, nil
			}
			inflight, inflightApp := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json"}, "")
			defer inflightApp.ReleaseCtx(inflight)
			inflight.Request().Header.SetCookie(auth.SessionCookieName, id)
			done := make(chan error, 1)
			go func() { done <- CheckRoute(store)(inflight) }()
			<-entered
			logout, logoutApp := createTestContext("POST", "/_logout", nil, "")
			defer logoutApp.ReleaseCtx(logout)
			logout.Request().Header.SetCookie(auth.SessionCookieName, id)
			require.NoError(t, LogoutRoute(store)(logout))
			require.Equal(t, 200, logout.Response().StatusCode())
			require.Empty(t, responseCookieValue(logout, auth.SessionCookieName))
			close(resume)
			require.NoError(t, <-done)
			require.Equal(t, 500, inflight.Response().StatusCode())
			require.Empty(t, inflight.GetRespHeader("X-Auth-User"))
			again, againApp := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json"}, "")
			defer againApp.ReleaseCtx(again)
			again.Request().Header.SetCookie(auth.SessionCookieName, id)
			require.NoError(t, CheckRoute(store)(again))
			require.Equal(t, 401, again.Response().StatusCode())
		})
	}
}

func TestSensitiveRefreshSaveFailureDoesNotAuthorize(t *testing.T) {
	setupRefreshTest(t)
	memory := sessionkit.NewMemoryStorage("sensitive:", 0)
	t.Cleanup(func() { require.NoError(t, memory.Close()) })
	cfg := sessionfiber.SessionConfig(sessionkit.NewManager(memory, sessionkit.DefaultConfig().WithCookieName(auth.SessionCookieName)))
	storage := &failSetStorage{Storage: cfg.Storage}
	cfg.Storage = storage
	store := session.NewStore(cfg)
	ctx, app := sensitiveTestSession(t, store, "POST", "/totp/enroll", "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return &warden.AllowListUser{UserID: "u1", Mail: "user@example.com", Status: "active"}, time.Time{}, nil
	}
	storage.fail = true
	allowed, err := authorizeSensitiveSession(ctx, sess)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, 500, ctx.Response().StatusCode())
	storage.fail = false
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	ctx.SetContext(canceled)
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return nil, time.Time{}, errors.New("dependency stopped")
	}
	allowed, err = authorizeSensitiveSession(ctx, sess)
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, 503, ctx.Response().StatusCode())
}
