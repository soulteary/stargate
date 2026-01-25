package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// IndexRoute handles GET requests to the root path (/).
// It checks if the user is authenticated and returns a simple status message.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func IndexRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}

		if !auth.IsAuthenticated(sess) {
			return ctx.SendString("Not authenticated")
		}

		return ctx.SendString("Authenticated")
	}
}
