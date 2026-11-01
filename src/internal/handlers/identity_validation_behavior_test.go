package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestWardenRejectsMalformedPhoneBeforeLookup(t *testing.T) {
	ctx, app := createTestContext("POST", "/_login", nil, "")
	defer app.ReleaseCtx(ctx)
	identity := &loginIdentity{userPhone: "not-a-phone"}
	_, span := noop.NewTracerProvider().Tracer("test").Start(context.Background(), "login")
	defer span.End()
	ok, err := identity.authenticateWarden(ctx, context.Background(), span)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, 400, ctx.Response().StatusCode())
}

func TestLegacyIdentityFallbackPreservesContactFields(t *testing.T) {
	ctx, app := createTestContext("POST", "/_login", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := setupTestStore().Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	storeLoginIdentity(sess, "warden", &loginIdentity{userID: "user-1", userPhone: "13800138000", userMail: "user@example.com"})
	require.Equal(t, "13800138000", sess.Get("user_phone"))
	require.Equal(t, "user@example.com", sess.Get("user_mail"))
	require.NotEqual(t, generateUserID("13800138000", ""), generateUserID("", "user@example.com"))
}

func TestRefreshLegacySessionUsesDefaultIntervalAndRejectsMissingIdentity(t *testing.T) {
	setupRefreshTest(t)
	config.AuthRefreshInterval.Value = ""
	ctx, app := createTestContext("GET", "/_auth", nil, "")
	defer app.ReleaseCtx(ctx)
	sess, err := setupTestStore().Get(ctx)
	require.NoError(t, err)
	defer sess.Release()
	sess.Set("auth_method", "warden")
	require.NoError(t, auth.Authenticate(sess))
	refreshed, err := refreshAuthorizationIfNeeded(context.Background(), sess)
	require.ErrorContains(t, err, "no refresh identity")
	require.False(t, refreshed)
	sess.Set("user_mail", "user@example.com")
	sess.Set("auth_refreshed_at", time.Now().Unix())
	require.False(t, wardenSessionNeedsRefresh(sess))
}
