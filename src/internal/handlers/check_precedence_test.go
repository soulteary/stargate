package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
)

func TestCheckRouteRefreshesOnlySelectedSession(t *testing.T) {
	for _, tc := range []struct {
		name, password, mail, directory   string
		trusted, protected                bool
		status, refreshCalls, headerCalls int
		user                              string
	}{
		{name: "password with unavailable old identity", password: "test123", directory: "unavailable", status: 200},
		{name: "password with deleted old identity", password: "test123", directory: "deleted", status: 200},
		{name: "trusted identity replaces old identity", trusted: true, mail: "new@example.test", directory: "unavailable", status: 200, headerCalls: 1, user: "new-user"},
		{name: "trusted lookup unavailable does not fall back", trusted: true, mail: "unavailable@example.test", directory: "active", status: 503, headerCalls: 1},
		{name: "trusted lookup deleted does not fall back", trusted: true, mail: "deleted@example.test", directory: "active", status: 401, headerCalls: 1},
		{name: "trusted credential without identity selects session", trusted: true, directory: "unavailable", status: 503, refreshCalls: 1},
		{name: "wrong password falls back to active session", password: "wrong", directory: "active", status: 200, refreshCalls: 1, user: "old-user"},
		{name: "wrong password fallback fails closed", password: "wrong", directory: "unavailable", status: 503, refreshCalls: 1},
		{name: "session deleted", directory: "deleted", status: 401, refreshCalls: 1},
		{name: "password cannot borrow verified session", password: "test123", directory: "unavailable", protected: true, status: 403},
		{name: "trusted identity cannot borrow verified session", trusted: true, mail: "new@example.test", directory: "unavailable", protected: true, status: 403, headerCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", "plaintext:test123")
			t.Setenv("PASSWORD_HEADER_AUTH_ENABLED", "true")
			t.Setenv("WARDEN_ENABLED", "true")
			t.Setenv("WARDEN_URL", "http://127.0.0.1:1")
			t.Setenv("AUTH_REFRESH_ENABLED", "true")
			t.Setenv("AUTH_REFRESH_INTERVAL", "1s")
			t.Setenv("HEADER_AUTH_ENABLED", "true")
			t.Setenv("HEADER_AUTH_SHARED_SECRET", "synthetic-trusted-proxy-secret-123456789")
			t.Setenv("STEP_UP_ENABLED", "false")
			if tc.protected {
				t.Setenv("STEP_UP_ENABLED", "true")
			}
			t.Setenv("STEP_UP_PATHS", "/admin")
			t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
			log := testLoggerCheckHeaders()
			if err := config.Initialize(log); err != nil {
				t.Fatal(err)
			}
			oldHandler, oldRefresh, oldHeader := forwardAuthHandler, lookupRefreshUser, lookupTrustedHeaderUser
			t.Cleanup(func() {
				forwardAuthHandler = oldHandler
				lookupRefreshUser = oldRefresh
				lookupTrustedHeaderUser = oldHeader
			})
			InitForwardAuthHandler(log)
			refreshCalls, headerCalls := 0, 0
			lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
				refreshCalls++
				if tc.directory == "unavailable" {
					return nil, time.Time{}, errors.New("synthetic outage")
				}
				if tc.directory == "deleted" {
					return nil, time.Time{}, auth.ErrUserUnavailable
				}
				return &warden.AllowListUser{UserID: "old-user", Mail: "old@example.test", Scope: []string{"updated"}, Role: "reader"}, time.Time{}, nil
			}
			lookupTrustedHeaderUser = func(_ context.Context, _, mail string) (*warden.AllowListUser, time.Time, error) {
				headerCalls++
				if mail == "unavailable@example.test" {
					return nil, time.Time{}, errors.New("synthetic outage")
				}
				if mail == "deleted@example.test" {
					return nil, time.Time{}, auth.ErrUserUnavailable
				}
				return &warden.AllowListUser{UserID: "new-user", Mail: mail}, time.Now().Add(time.Minute), nil
			}
			store := setupTestStore()
			ctx, app := createTestContext("GET", "/_auth", nil, "")
			defer app.ReleaseCtx(ctx)
			sess, err := store.Get(ctx)
			if err != nil {
				t.Fatal(err)
			}
			sess.Set("auth_method", "warden")
			sess.Set("user_id", "old-user")
			sess.Set("user_mail", "old@example.test")
			sess.Set("auth_refreshed_at", time.Now().Add(-time.Minute).Unix())
			sess.Set("step_up_verified_at", time.Now().Unix())
			if err := auth.Authenticate(sess); err != nil {
				t.Fatal(err)
			}
			next := rebindSessionContext(t, app, ctx, sess)
			defer app.ReleaseCtx(next)
			next.Request().Header.Set("Accept", "application/json")
			if tc.password != "" {
				next.Request().Header.Set("Stargate-Password", tc.password)
			}
			if tc.trusted {
				next.Request().Header.Set("X-Stargate-Header-Auth", config.HeaderAuthSharedSecret.String())
			}
			if tc.mail != "" {
				next.Request().Header.Set("X-User-Mail", tc.mail)
			}
			if tc.protected {
				next.Request().Header.Set("X-Forwarded-Uri", "/admin/settings")
			}
			if err := CheckRoute(store)(next); err != nil {
				t.Fatal(err)
			}
			if got := next.Response().StatusCode(); got != tc.status {
				t.Fatalf("status=%d want%d body=%s", got, tc.status, next.Response().Body())
			}
			if refreshCalls != tc.refreshCalls || headerCalls != tc.headerCalls {
				t.Fatalf("lookups refresh=%d header=%d", refreshCalls, headerCalls)
			}
			if tc.user != "" && string(next.Response().Header.Peek("X-Auth-User")) != tc.user {
				t.Fatalf("unexpected user %s", next.Response().Header.Peek("X-Auth-User"))
			}
			if tc.user == "old-user" && string(next.Response().Header.Peek("X-Auth-Scopes")) != "updated" {
				t.Fatal("refreshed claims were not published")
			}
			if tc.status != fiber.StatusOK && len(next.Response().Header.Peek("X-Auth-User")) != 0 {
				t.Fatal("failed authentication published identity")
			}
		})
	}
}
