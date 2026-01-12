package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// LogoutRoute handles GET requests to /_logout for user logout.
// It destroys the user's session and returns a confirmation message.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LogoutRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		err = auth.Unauthenticate(sess)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.authenticate_failed"))
		}

		return ctx.SendString("Logged out")
	}
}
