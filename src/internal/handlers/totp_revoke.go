package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// TOTPRevokeRoute handles GET /totp/revoke - shows TOTP unbind confirm page (requires auth).
func TOTPRevokeRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}
		if !auth.IsAuthenticated(sess) {
			return ctx.Redirect("/_login", fiber.StatusFound)
		}
		client := getHeraldTOTPClient()
		if client == nil {
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable"))
		}
		userID, _ := sess.Get("user_id").(string)
		if userID == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, "user_id not in session")
		}
		return ctx.Render("totp_revoke", fiber.Map{
			"Title":             config.LoginPageTitle.Value,
			"FooterText":        config.LoginPageFooterText.Value,
			"HeraldTOTPEnabled": config.HeraldTOTPEnabled.ToBool(),
		})
	}
}

// TOTPRevokeConfirmAPI handles POST /totp/revoke - revokes TOTP for current user (requires auth).
func TOTPRevokeConfirmAPI(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}
		if !auth.IsAuthenticated(sess) {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"ok": false, "error": "unauthorized"})
		}
		client := getHeraldTOTPClient()
		if client == nil {
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"ok": false, "error": "TOTP service unavailable"})
		}
		userID, _ := sess.Get("user_id").(string)
		if userID == "" {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"ok": false, "error": "user_id not in session"})
		}
		_, err = client.Revoke(ctx.Context(), userID)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("TOTP revoke failed")
			return ctx.Status(fiber.StatusBadGateway).JSON(fiber.Map{"ok": false, "error": "revoke_failed"})
		}
		return ctx.JSON(fiber.Map{"ok": true, "subject": userID})
	}
}
