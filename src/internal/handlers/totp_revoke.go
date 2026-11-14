package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auditlog"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// revokeErrorReason maps herald-totp revoke error to a frontend-safe reason code.
func revokeErrorReason(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "401") || strings.Contains(s, "unauthorized"):
		return "unauthorized"
	case strings.Contains(s, "404") || strings.Contains(s, "Cannot POST"):
		return "unavailable"
	case strings.Contains(s, "429") || strings.Contains(s, "rate_limit"):
		return "rate_limited"
	case strings.Contains(s, "502") || strings.Contains(s, "503") || strings.Contains(s, "connection refused") || strings.Contains(s, "timeout"):
		return "service_unavailable"
	default:
		return "service_error"
	}
}

// revokeNotConfirmed reports whether the Herald SDK returned its
// "revoke_not_confirmed" error: Herald answered 200 without confirming the
// revocation. SDK releases before v1.6.0 returned no error in that case.
func revokeNotConfirmed(err error) bool {
	var heraldErr *herald.HeraldError
	return errors.As(err, &heraldErr) && heraldErr.Reason == "revoke_not_confirmed"
}

// TOTPRevokeRoute handles GET /totp/revoke - shows TOTP unbind confirm page (requires auth).
func TOTPRevokeRoute(store *session.Store) func(c fiber.Ctx) error {
	return func(ctx fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}
		defer sess.Release()
		if !auth.IsAuthenticated(sess) {
			return ctx.Redirect().Status(fiber.StatusFound).To("/_login")
		}
		if allowed, err := authorizeSensitiveSession(ctx, store, sess); !allowed {
			return err
		}
		client := getHeraldClient()
		if client == nil {
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable"))
		}
		userID, _ := sess.Get("user_id").(string)
		if userID == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, "user_id not in session")
		}
		proof, failure := requiredReauthProof(ctx.Context(), sess)
		if failure != nil {
			return SendErrorResponse(ctx, failure.status, failure.message)
		}
		ctx.Set("Cache-Control", "no-store")
		uiWords, uiJSON := authenticationUIData(ctx)
		return ctx.Render("totp_revoke", reauthView(fiber.Map{
			"Title":             config.LoginPageTitle.Value,
			"UI":                uiWords,
			"PageTitle":         uiWords["revoke"],
			"UIMessages":        uiJSON,
			"UILanguage":        authenticationUILanguage(ctx),
			"SupportURL":        config.LoginSupportURL.String(),
			"FooterText":        config.LoginPageFooterText.Value,
			"HeraldTOTPEnabled": config.HeraldTOTPEnabled.ToBool(),
		}, proof))
	}
}

// TOTPRevokeConfirmAPI handles POST /totp/revoke - revokes TOTP for current user (requires auth).
func TOTPRevokeConfirmAPI(store *session.Store) func(c fiber.Ctx) error {
	return func(ctx fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}
		defer sess.Release()
		if !auth.IsAuthenticated(sess) {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"ok": false, "error": "unauthorized"})
		}
		if allowed, err := authorizeSensitiveSession(ctx, store, sess); !allowed {
			return err
		}
		client := getHeraldClient()
		if client == nil {
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"ok": false, "error": "TOTP service unavailable"})
		}
		userID, _ := sess.Get("user_id").(string)
		if userID == "" {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"ok": false, "error": "user_id not in session"})
		}
		// The proof is the user's own factor. The shared password is not
		// accepted for a session that carries a user identity.
		if failure := reauthenticate(ctx, sess); failure != nil {
			if failure == reauthFailed {
				auditlog.LogTOTPChange(ctx.Context(), userID, auditlog.TOTPActionRevoked, ctx.IP(), false, "reauthentication_failed")
			}
			return ctx.Status(failure.status).JSON(fiber.Map{"ok": false, "error": failure.code})
		}
		// The proof given here removes TOTP and nothing else. A proof stored
		// for enrollment must not carry over to the state after the removal.
		discardEnrollmentProof(sess)
		if err := saveLiveSession(ctx.Context(), store, sess); err != nil {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"ok": false, "error": "session_save_failed"})
		}
		revokeResp, err := client.TOTPRevoke(ctx.Context(), userID)
		notConfirmed := revokeNotConfirmed(err)
		if err != nil && !notConfirmed {
			log.Warn().Err(err).Str("user_id", userID).Msg("TOTP revoke failed")
			reason := revokeErrorReason(err)
			auditlog.LogTOTPChange(ctx.Context(), userID, auditlog.TOTPActionRevoked, ctx.IP(), false, reason)
			return ctx.Status(fiber.StatusBadGateway).JSON(fiber.Map{"ok": false, "error": "revoke_failed", "reason": reason})
		}
		// Herald can answer 200 without confirming the revocation. Reporting
		// success then would tell the user the factor is gone when it is not.
		if notConfirmed || revokeResp == nil || !revokeResp.OK {
			log.Warn().Str("user_id", userID).Msg("TOTP revoke was not confirmed by Herald")
			auditlog.LogTOTPChange(ctx.Context(), userID, auditlog.TOTPActionRevoked, ctx.IP(), false, "revoke_not_confirmed")
			return ctx.Status(fiber.StatusBadGateway).JSON(fiber.Map{"ok": false, "error": "revoke_failed", "reason": "revoke_not_confirmed"})
		}
		auditlog.LogTOTPChange(ctx.Context(), userID, auditlog.TOTPActionRevoked, ctx.IP(), true, "")
		return ctx.JSON(fiber.Map{"ok": true, "subject": userID})
	}
}
