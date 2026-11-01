package handlers

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/stretchr/testify/require"
)

func TestLoginSessionIDRecoversCookieOrStoreAndRejectsFailure(t *testing.T) {
	for _, cookie := range []string{auth.SessionCookieName + "=cookie-id; Path=/", "other=value; " + auth.SessionCookieName + "=cookie-id"} {
		ctx, app := createTestContext("POST", "/_login", nil, "")
		ctx.Response().Header.Set("Set-Cookie", cookie)
		id, err := loginSessionID(ctx, &session.Session{}, &MockSessionGetter{})
		require.NoError(t, err)
		require.Equal(t, "cookie-id", id)
		app.ReleaseCtx(ctx)
	}
	for _, fail := range []bool{false, true} {
		ctx, app := createTestContext("POST", "/_login", nil, "")
		ctx.Response().Header.Set("Set-Cookie", "other=value")
		store := setupTestStore()
		var want string
		getter := &MockSessionGetter{GetFunc: func(c fiber.Ctx) (*session.Session, error) {
			if fail {
				return nil, errors.New("injected session read failure")
			}
			sess, err := store.Get(c)
			if sess != nil {
				want = sess.ID()
			}
			return sess, err
		}}
		id, err := loginSessionID(ctx, &session.Session{}, getter)
		require.NoError(t, err)
		if fail {
			require.Empty(t, id)
			require.Equal(t, 500, ctx.Response().StatusCode())
		} else {
			require.NotEmpty(t, id)
			require.Equal(t, want, id)
		}
		app.ReleaseCtx(ctx)
	}
	ctx, app := createTestContext("POST", "/_login", nil, "")
	defer app.ReleaseCtx(ctx)
	getter := &MockSessionGetter{GetFunc: func(fiber.Ctx) (*session.Session, error) { return &session.Session{}, nil }}
	id, err := loginSessionID(ctx, &session.Session{}, getter)
	require.NoError(t, err)
	require.Empty(t, id)
	require.Equal(t, 500, ctx.Response().StatusCode())
}
