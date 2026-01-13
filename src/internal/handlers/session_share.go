package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

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
	return func(ctx *fiber.Ctx) error {
		sessionID := ctx.Query("id")
		if sessionID == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T("error.missing_session_id"))
		}

		cookie := &fiber.Cookie{
			Name:     auth.SessionCookieName,
			Value:    sessionID,
			Expires:  time.Now().Add(config.SessionExpiration),
			SameSite: fiber.CookieSameSiteLaxMode,
			HTTPOnly: true,
		}

		// If Cookie domain is configured, set it
		if config.CookieDomain.Value != "" {
			cookie.Domain = config.CookieDomain.Value
		}

		ctx.Cookie(cookie)

		return ctx.Redirect("/")
	}
}
