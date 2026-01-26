package handlers

import (
	"github.com/gofiber/fiber/v2"

	session "github.com/soulteary/session-kit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// SessionShareRoute handles GET requests to /_session_exchange for cross-domain session sharing.
// It sets a session cookie with the provided session ID and redirects to the root path.
// This allows sessions to be shared across different domains/subdomains.
//
// Query parameters:
//   - id: Session ID to set in the cookie
//
// Returns a Fiber handler function.
func SessionShareRoute() func(c *fiber.Ctx) error {
	// Create session config for cookie creation
	sessionConfig := session.DefaultConfig().
		WithCookieName(auth.SessionCookieName).
		WithExpiration(config.SessionExpiration).
		WithCookieDomain(config.CookieDomain.Value).
		WithSameSite("Lax").
		WithHTTPOnly(true)

	return func(ctx *fiber.Ctx) error {
		sessionID := ctx.Query("id")
		if sessionID == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.missing_session_id"))
		}

		// Use session-kit's CreateCookie for consistent cookie creation
		cookie := session.CreateCookie(sessionConfig, sessionID)
		ctx.Cookie(cookie)

		return ctx.Redirect("/")
	}
}
