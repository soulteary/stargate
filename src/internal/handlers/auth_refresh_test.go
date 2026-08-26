package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
)

func setupRefreshTest(t *testing.T) {
	t.Helper()
	originalEnabled := config.AuthRefreshEnabled.Value
	originalInterval := config.AuthRefreshInterval.Value
	originalLookup := lookupRefreshUser
	originalWardenEnabled := config.WardenEnabled.Value
	t.Cleanup(func() {
		config.AuthRefreshEnabled.Value = originalEnabled
		config.AuthRefreshInterval.Value = originalInterval
		lookupRefreshUser = originalLookup
		config.WardenEnabled.Value = originalWardenEnabled
	})
	config.AuthRefreshEnabled.Value = "true"
	config.AuthRefreshInterval.Value = "1s"
	config.WardenEnabled.Value = "true"
}

func TestAuthRefreshRejectsMissingOrDisabledUser(t *testing.T) {
	setupRefreshTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_mail", "disabled@example.com")
	sess.Set("auth_refreshed_at", time.Now().Add(-time.Minute).Unix())
	testza.AssertNoError(t, auth.Authenticate(sess))
	lookupRefreshUser = func(context.Context, string, string) *warden.AllowListUser {
		return &warden.AllowListUser{Status: "disabled"}
	}

	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	testza.AssertNotNil(t, err)
	testza.AssertFalse(t, refreshed)
	testza.AssertFalse(t, auth.IsAuthenticated(sess))
}

func TestAuthRefreshReplacesRevokedAuthorization(t *testing.T) {
	setupRefreshTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_mail", "active@example.com")
	sess.Set("user_scope", []string{"old-admin"})
	sess.Set("auth_refreshed_at", time.Now().Add(-time.Minute).Unix())
	lookupRefreshUser = func(context.Context, string, string) *warden.AllowListUser {
		return &warden.AllowListUser{
			UserID: "user-1",
			Mail:   "active@example.com",
			Status: "active",
			Scope:  []string{"read"},
			Role:   "viewer",
		}
	}

	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, refreshed)
	testza.AssertEqual(t, "viewer", sess.Get("user_role"))
	testza.AssertEqual(t, []string{"read"}, sess.Get("user_scope"))
}

func TestAuthRefreshSkipsAnonymousSession(t *testing.T) {
	setupRefreshTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	lookupCalled := false
	lookupRefreshUser = func(context.Context, string, string) *warden.AllowListUser {
		lookupCalled = true
		return nil
	}

	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, refreshed)
	testza.AssertFalse(t, lookupCalled)
}

func TestAuthRefreshSkipsStandaloneSession(t *testing.T) {
	setupRefreshTest(t)
	config.WardenEnabled.Value = "false"

	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, auth.Authenticate(sess))

	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, refreshed)
}
