package handlers

import (
	"errors"
	"testing"

	forwardauth "github.com/soulteary/forwardauth-kit/v3"
	fafiber "github.com/soulteary/forwardauth-kit/v3/fiberadapter"
	"github.com/stretchr/testify/require"
)

func TestForwardAuthFailuresRejectAPIRequests(t *testing.T) {
	for _, err := range []error{forwardauth.ErrInvalidPassword, forwardauth.ErrNotAuthenticated, forwardauth.ErrUserNotFound, forwardauth.ErrSessionRequired, errors.New("backend failure")} {
		ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": "application/json"}, "")
		require.NoError(t, handleForwardAuthFailure(ctx, GetForwardAuthHandler(), fafiber.NewContext(ctx), err))
		require.Equal(t, 401, ctx.Response().StatusCode())
		app.ReleaseCtx(ctx)
	}
}
