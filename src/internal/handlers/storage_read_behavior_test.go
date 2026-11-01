package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/stretchr/testify/require"
)

type unavailableSessionStorage struct{ fiber.Storage }

func (*unavailableSessionStorage) Get(string) ([]byte, error) {
	return nil, errors.New("injected storage read failure")
}
func (*unavailableSessionStorage) GetWithContext(context.Context, string) ([]byte, error) {
	return nil, errors.New("injected storage read failure")
}

func TestSensitiveRoutesRejectSessionStorageReadFailure(t *testing.T) {
	store := session.NewStore(session.Config{Storage: &unavailableSessionStorage{}, Extractor: extractors.FromCookie(auth.SessionCookieName)})
	for _, tc := range []struct {
		name    string
		handler fiber.Handler
		want    int
	}{
		{"TOTP page", TOTPEnrollPageRoute(store), 500},
		{"TOTP start", TOTPEnrollRoute(store), 500},
		{"TOTP confirm", TOTPEnrollConfirmAPI(store), 500},
		{"TOTP revoke page", TOTPRevokeRoute(store), 500},
		{"TOTP revoke", TOTPRevokeConfirmAPI(store), 500},
		{"step-up page", StepUpRoute(store), 401},
		{"step-up verify", StepUpAPI(store), 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, app := createTestContext("POST", "/", nil, "")
			defer app.ReleaseCtx(ctx)
			ctx.Request().Header.SetCookie(auth.SessionCookieName, "d2dad012-0af7-44cd-86a7-dc04f69878a6")
			require.NoError(t, tc.handler(ctx))
			require.Equal(t, tc.want, ctx.Response().StatusCode())
			require.NotContains(t, string(ctx.Response().Body()), "injected storage read failure")
		})
	}
}
