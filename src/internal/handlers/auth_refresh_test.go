package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
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
	sess.Set("auth_method", "warden")
	sess.Set("auth_refreshed_at", time.Now().Add(-time.Minute).Unix())
	testza.AssertNoError(t, auth.Authenticate(sess))
	// Authenticate rotates the session id, so the refresh has to run against a
	// context carrying the new one, the way the next real request would.
	ctx = rebindSessionContext(t, app, ctx, sess)
	defer app.ReleaseCtx(ctx)
	sess, err = store.Get(ctx)
	testza.AssertNoError(t, err)
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return &warden.AllowListUser{Status: "disabled"}, time.Time{}, nil
	}

	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	testza.AssertNotNil(t, err)
	testza.AssertFalse(t, refreshed)
	testza.AssertFalse(t, auth.IsAuthenticated(sess))
}

func TestAuthRefreshPreservesSessionWhenRequestDeadlineExpires(t *testing.T) {
	setupRefreshTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_mail", "active@example.com")
	sess.Set("auth_method", "warden")
	sess.Set("auth_refreshed_at", time.Now().Add(-time.Minute).Unix())
	testza.AssertNoError(t, auth.Authenticate(sess))

	lookupRefreshUser = func(ctx context.Context, _, _ string) (*warden.AllowListUser, time.Time, error) {
		<-ctx.Done()
		return nil, time.Time{}, nil
	}
	requestCtx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-requestCtx.Done()

	refreshed, err := refreshAuthorizationIfNeeded(requestCtx, sess)
	testza.AssertEqual(t, context.DeadlineExceeded, err)
	testza.AssertFalse(t, refreshed)
	testza.AssertTrue(t, auth.IsAuthenticated(sess))
}

func TestAuthRefreshReplacesRevokedAuthorization(t *testing.T) {
	setupRefreshTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("user_mail", "active@example.com")
	sess.Set("user_id", "user-1")
	sess.Set("auth_method", "warden")
	sess.Set("user_scope", []string{"old-admin"})
	sess.Set("auth_refreshed_at", time.Now().Add(-time.Minute).Unix())
	testza.AssertNoError(t, auth.Authenticate(sess))
	// Authenticate rotates the session id, so the refresh has to run against a
	// context carrying the new one, the way the next real request would.
	ctx = rebindSessionContext(t, app, ctx, sess)
	defer app.ReleaseCtx(ctx)
	sess, err = store.Get(ctx)
	testza.AssertNoError(t, err)
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		return &warden.AllowListUser{
			UserID: "user-1",
			Mail:   "active@example.com",
			Status: "active",
			Scope:  []string{"read"},
			Role:   "viewer",
		}, time.Time{}, nil
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
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		lookupCalled = true
		return nil, time.Time{}, nil
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
	sessionID := sess.ID()
	testza.AssertNoError(t, auth.Authenticate(sess))
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sessionID)
	sess, err = store.Get(ctx)
	testza.AssertNoError(t, err)

	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, refreshed)
}

func TestAuthRefreshSkipsPasswordSessionInMixedDeployment(t *testing.T) {
	setupRefreshTest(t)
	store := setupTestStore()
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := store.Get(ctx)
	testza.AssertNoError(t, err)
	sess.Set("auth_method", "password")
	testza.AssertNoError(t, auth.Authenticate(sess))

	lookupCalled := false
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		lookupCalled = true
		return nil, time.Time{}, nil
	}
	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, refreshed)
	testza.AssertFalse(t, lookupCalled)
}

// The interval is measured on authorizationNow alone: the clock here stands
// in 2001 and only moves when the test moves it, so the outcome cannot depend
// on the wall clock or on where in a second the test runs.
func TestAuthRefreshIntervalFollowsAuthorizationClock(t *testing.T) {
	isolateGlobalState(t)
	setupRefreshTest(t)
	previousNow := authorizationNow
	t.Cleanup(func() { authorizationNow = previousNow })
	config.AuthRefreshInterval.Value = "5m"
	current := time.Unix(1_000_000_000, 0)
	authorizationNow = func() time.Time { return current }

	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := setupTestStore().Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	sess.Set("auth_method", "warden")
	sess.Set("user_id", "user-1")
	sess.Set("user_mail", "active@example.com")
	require.NoError(t, auth.Authenticate(sess))
	lookups := 0
	lookupRefreshUser = func(context.Context, string, string) (*warden.AllowListUser, time.Time, error) {
		lookups++
		return &warden.AllowListUser{UserID: "user-1", Mail: "active@example.com", Status: "active"}, time.Time{}, nil
	}

	// A session without a stamp is always due, and the stamp is taken from
	// the same clock.
	stampedAt := current
	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	require.NoError(t, err)
	require.True(t, refreshed)
	require.Equal(t, stampedAt.Unix(), sess.Get("auth_refreshed_at"))

	for _, step := range []struct {
		name    string
		age     time.Duration
		refresh bool
	}{
		{"same instant", 0, false},
		{"inside the interval", 5*time.Minute - time.Second, false},
		{"exactly the interval", 5 * time.Minute, false},
		{"past the interval", 5*time.Minute + time.Second, true},
	} {
		before := lookups
		current = stampedAt.Add(step.age)
		refreshed, err = refreshAuthorizationIfNeeded(context.Background(), sess)
		require.NoError(t, err, step.name)
		require.Equal(t, step.refresh, refreshed, step.name)
		if step.refresh {
			require.Equal(t, before+1, lookups, step.name)
			require.Equal(t, current.Unix(), sess.Get("auth_refreshed_at"), step.name)
		} else {
			require.Equal(t, before, lookups, step.name)
			require.Equal(t, stampedAt.Unix(), sess.Get("auth_refreshed_at"), step.name)
		}
	}

	// The new stamp starts a new interval on the same clock.
	require.False(t, wardenSessionNeedsRefresh(sess))
	current = current.Add(5*time.Minute + time.Second)
	require.True(t, wardenSessionNeedsRefresh(sess))
}
