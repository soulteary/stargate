package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/oidc"
)

var (
	oidcProvider     *oidc.Provider
	oidcStateManager *oidc.StateManager
)

// InitOIDC initializes the OIDC provider and state manager
func InitOIDC() error {
	if !config.IsOIDCEnabled() {
		return nil
	}

	redirectURI := config.OIDCRedirectURI.Value
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("https://%s/_oidc/callback", config.AuthHost.Value)
	}

	var err error
	oidcProvider, err = oidc.NewProvider(
		config.OIDCIssuerURL.Value,
		config.OIDCClientID.Value,
		config.OIDCClientSecret.Value,
		redirectURI,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	oidcStateManager = oidc.NewStateManager()
	return nil
}

// OIDCLoginHandler initiates the OIDC login flow
func OIDCLoginHandler(store *session.Store) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if oidcProvider == nil {
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T("error.oidc_not_configured"))
		}

		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		state, err := oidcStateManager.GenerateState()
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.state_generation_failed"))
		}

		if err := oidcStateManager.SetState(sess, state); err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		authURL := oidcProvider.AuthURL(state)
		return ctx.Redirect(authURL)
	}
}

// OIDCCallbackHandler handles the OIDC callback
func OIDCCallbackHandler(store *session.Store) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if oidcProvider == nil {
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T("error.oidc_not_configured"))
		}

		sess, err := store.Get(ctx)
		if err != nil {
			return renderOIDCErrorPage(ctx, i18n.T("error.session_store_failed"))
		}

		code := ctx.Query("code")
		state := ctx.Query("state")

		if code == "" {
			return renderOIDCErrorPage(ctx, i18n.T("error.oidc_missing_code"))
		}

		if !oidcStateManager.ValidateState(sess, state) {
			return renderOIDCErrorPage(ctx, i18n.T("error.oidc_invalid_state"))
		}

		token, err := oidcProvider.Exchange(context.Background(), code)
		if err != nil {
			return renderOIDCErrorPage(ctx, i18n.T("error.oidc_token_exchange_failed"))
		}

		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			return renderOIDCErrorPage(ctx, i18n.T("error.oidc_missing_id_token"))
		}

		userInfo, err := oidcProvider.GetUserInfoFromToken(context.Background(), rawIDToken)
		if err != nil {
			return renderOIDCErrorPage(ctx, i18n.T("error.oidc_token_verification_failed"))
		}

		if err := auth.AuthenticateOIDC(sess, userInfo.UserID, userInfo.Email); err != nil {
			return renderOIDCErrorPage(ctx, i18n.T("error.authenticate_failed"))
		}

		callbackURL := ctx.Query("callback")
		if callbackURL == "" {
			callbackURL = fmt.Sprintf("/_session_exchange?id=%s", sess.ID())
		} else {
			callbackURL = fmt.Sprintf("%s/_session_exchange?id=%s", callbackURL, sess.ID())
		}

		return ctx.Redirect(callbackURL)
	}
}

// renderOIDCErrorPage renders an error page with retry button
func renderOIDCErrorPage(ctx *fiber.Ctx, message string) error {
	if IsHTMLRequest(ctx) {
		return ctx.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        body { font-family: system-ui, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #e74c3c; }
        a { display: inline-block; margin-top: 1rem; padding: 0.5rem 1rem; background: #3498db; color: white; text-decoration: none; border-radius: 4px; }
        a:hover { background: #2980b9; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Authentication Error</h1>
        <p>%s</p>
        <a href="/_login">Retry</a>
    </div>
</body>
</html>
		`, i18n.T("error.oidc_error"), message))
	}

	return SendErrorResponse(ctx, fiber.StatusBadRequest, message)
}
