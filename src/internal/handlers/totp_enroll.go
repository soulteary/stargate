package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/heraldtotp"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// TOTPEnrollRoute handles GET /totp/enroll - shows TOTP bind page (requires auth).
// Calls herald-totp enroll/start and renders page with QR (otpauth_uri) and enroll_id.
func TOTPEnrollRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}
		if !auth.IsAuthenticated(sess) {
			return ctx.Redirect("/_login", fiber.StatusFound)
		}
		userID, _ := sess.Get("user_id").(string)
		if userID == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, "user_id not in session")
		}
		label, _ := sess.Get("user_mail").(string)
		if label == "" {
			label, _ = sess.Get("user_phone").(string)
		}
		if label == "" {
			label = userID
		}

		client := getHeraldTOTPClient()
		if client == nil {
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable"))
		}
		startResp, err := client.EnrollStart(ctx.Context(), &heraldtotp.EnrollStartRequest{
			Subject: userID,
			Label:   label,
		})
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("TOTP enroll start failed")
			return SendErrorResponse(ctx, fiber.StatusBadGateway, "TOTP enroll start failed")
		}
		return ctx.Render("totp_enroll", fiber.Map{
			"Title":             config.LoginPageTitle.Value,
			"FooterText":        config.LoginPageFooterText.Value,
			"EnrollID":          startResp.EnrollID,
			"OtpauthURI":        startResp.OtpauthURI,
			"HeraldTOTPEnabled": config.HeraldTOTPEnabled.ToBool(),
		})
	}
}

// TOTPEnrollConfirmAPI handles POST /totp/enroll/confirm - confirms TOTP with code (requires auth).
func TOTPEnrollConfirmAPI(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}
		if !auth.IsAuthenticated(sess) {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"ok": false, "error": "unauthorized"})
		}
		enrollID := ctx.FormValue("enroll_id")
		code := ctx.FormValue("code")
		if enrollID == "" || code == "" {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"ok": false, "error": "enroll_id and code required"})
		}
		client := getHeraldTOTPClient()
		if client == nil {
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"ok": false, "error": "TOTP service unavailable"})
		}
		confirmResp, err := client.EnrollConfirm(ctx.Context(), &heraldtotp.EnrollConfirmRequest{
			EnrollID: enrollID,
			Code:     code,
		})
		if err != nil {
			log.Warn().Err(err).Str("enroll_id", enrollID).Msg("TOTP enroll confirm failed")
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"ok": false, "error": "invalid_code"})
		}
		return ctx.JSON(fiber.Map{
			"ok":           true,
			"subject":      confirmResp.Subject,
			"totp_enabled": confirmResp.TotpEnabled,
			"backup_codes": confirmResp.BackupCodes,
		})
	}
}
