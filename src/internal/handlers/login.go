package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// LoginAPI handles POST requests to /_login for password authentication.
// It validates the password from the form data, creates a session if valid,
// and returns 200 OK on success or 401/500 on failure.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginAPI(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		password := ctx.FormValue("password")

		if !auth.CheckPassword(password) {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
		}

		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		err = auth.Authenticate(sess)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.authenticate_failed"))
		}

		return ctx.SendStatus(fiber.StatusOK)
	}
}

// LoginRoute handles GET requests to /_login for displaying the login page.
// If the user is already authenticated, it redirects to the session exchange endpoint.
// Otherwise, it renders the login page template.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		callback := ctx.Query("callback")

		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		if auth.IsAuthenticated(sess) {
			// 使用 X-Forwarded-* 头部构建正确的重定向 URL
			proto := GetForwardedProto(ctx)
			if proto == "" {
				proto = ctx.Protocol()
			}
			redirectURL := fmt.Sprintf("%s://%s/_session_exchange?id=%s", proto, callback, sess.ID())
			return ctx.Redirect(redirectURL)
		}

		return ctx.Render("login", fiber.Map{
			"Callback":   callback,
			"SessionID":  sess.ID(),
			"Title":      config.LoginPageTitle.Value,
			"FooterText": config.LoginPageFooterText.Value,
		})
	}
}
