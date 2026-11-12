package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

func authenticationView(c fiber.Ctx, page string) fiber.Map {
	words, encoded := authenticationUIData(c)
	c.Set("Cache-Control", "no-store")
	return fiber.Map{"PageTitle": words[page], "UI": words, "UIMessages": encoded, "UILanguage": authenticationUILanguage(c), "SupportURL": config.LoginSupportURL.String(), "Title": config.LoginPageTitle.Value, "FooterText": config.LoginPageFooterText.Value, "HeraldTOTPEnabled": config.HeraldTOTPEnabled.ToBool()}
}

// AccountRoute renders session management without changing the root API response
// or creating enrollment state. The normal fresh-directory guard still applies.
func AccountRoute(store *session.Store) fiber.Handler {
	return func(c fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return SendErrorResponse(c, 500, i18n.T(c, "error.session_store_failed"))
		}
		defer sess.Release()
		if !auth.IsAuthenticated(sess) {
			return c.Redirect().Status(302).To("/_login")
		}
		if allowed, err := authorizeSensitiveSession(c, store, sess); !allowed {
			return err
		}
		return c.Render("account", authenticationView(c, "account"))
	}
}
