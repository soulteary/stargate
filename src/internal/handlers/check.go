package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// handleNotAuthenticated handles unauthenticated requests.
// For HTML requests, it redirects to the login page.
// For API requests (JSON/XML), it returns a 401 error response.
func handleNotAuthenticated(ctx *fiber.Ctx) error {
	if IsHTMLRequest(ctx) {
		// HTML request: redirect to login page
		callbackURL := BuildCallbackURL(ctx)
		return ctx.Redirect(callbackURL)
	}

	// Non-HTML request: return 401 error
	return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.auth_required"))
}

// CheckRoute is the main authentication check handler for Traefik Forward Auth.
// It validates requests in two ways:
//  1. Stargate-Password header authentication (for API requests)
//  2. Session cookie authentication (for web requests)
//
// On successful authentication, it sets the X-Forwarded-User header (or configured header name)
// and returns 200 OK. On failure, it either redirects to login (HTML) or returns 401 (API).
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func CheckRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			// Session store error, return 500 error
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		// Handle Stargate-Password Header authentication
		stargatePassword := ctx.Get("Stargate-Password")
		if stargatePassword != "" {
			if !auth.CheckPassword(stargatePassword) {
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
			}

			// Authentication successful, set user info header
			// Since Stargate uses password authentication, there's no specific username, use default value
			userHeaderName := config.UserHeaderName.String()
			ctx.Set(userHeaderName, "authenticated")
			return ctx.SendStatus(fiber.StatusOK)
		}

		// Check session authentication
		if !auth.IsAuthenticated(sess) {
			return handleNotAuthenticated(ctx)
		}

		// Authentication successful, set user info header
		userHeaderName := config.UserHeaderName.String()
		userValue := auth.GetForwardedUserValue(sess)
		ctx.Set(userHeaderName, userValue)

		return ctx.SendStatus(fiber.StatusOK)
	}
}
