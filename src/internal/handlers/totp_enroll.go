package handlers

import (
	"errors"
	"html/template"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auditlog"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

const (
	totpEnrollmentValidity   = 10 * time.Minute
	totpEnrollmentIDKey      = "totp_enrollment_id"
	totpEnrollmentSubjectKey = "totp_enrollment_subject"
	totpEnrollmentStartedKey = "totp_enrollment_started_at"
)

func clearTOTPEnrollment(sess *session.Session) {
	sess.Delete(totpEnrollmentIDKey)
	sess.Delete(totpEnrollmentSubjectKey)
	sess.Delete(totpEnrollmentStartedKey)
}

// TOTPEnrollPageRoute renders a safe confirmation page. Enrollment state is
// created only by the same-origin POST /totp/enroll action. The page asks for
// the session's re-authentication proof unless a recent one is on record.
func TOTPEnrollPageRoute(store *session.Store) func(c fiber.Ctx) error {
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
		proof := reauthProofNone
		if !reauthenticatedRecently(sess) {
			var failure *reauthFailure
			if proof, failure = requiredReauthProof(ctx.Context(), sess); failure != nil {
				return SendErrorResponse(ctx, failure.status, failure.message)
			}
		}
		return ctx.Render("totp_start", reauthView(authenticationView(ctx, "bind"), proof))
	}
}

// TOTPEnrollRoute handles POST /totp/enroll - starts enrollment and shows the bind page.
// Calls herald-totp enroll/start and renders page with QR (otpauth_uri) and enroll_id.
func TOTPEnrollRoute(store *session.Store) func(c fiber.Ctx) error {
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
		userID, _ := sess.Get("user_id").(string)
		if userID == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, "user_id not in session")
		}
		// A session cookie and a recent login are not enough to bind a new
		// factor: the request must carry the user's own proof, or follow one.
		if !reauthenticatedRecently(sess) {
			if failure := reauthenticate(ctx, sess); failure != nil {
				if failure == reauthFailed {
					auditlog.LogTOTPChange(ctx.Context(), userID, auditlog.TOTPActionEnrolled, ctx.IP(), false, "reauthentication_failed")
				}
				return SendErrorResponse(ctx, failure.status, failure.message)
			}
			if err := saveLiveSession(ctx.Context(), store, sess); err != nil {
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
			}
		}
		label, _ := sess.Get("user_mail").(string)
		if label == "" {
			label, _ = sess.Get("user_phone").(string)
		}
		if label == "" {
			label = userID
		}

		client := getHeraldClient()
		if client == nil {
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable"))
		}
		// If already bound TOTP, redirect to revoke page
		statusResp, err := client.TOTPStatus(ctx.Context(), userID)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("TOTP status check failed")
			return SendErrorResponse(ctx, fiber.StatusBadGateway, "TOTP status check failed")
		}
		if statusResp.TotpEnabled {
			return ctx.Redirect().Status(fiber.StatusFound).To("/totp/revoke")
		}
		startResp, err := client.TOTPEnrollStart(ctx.Context(), &herald.TOTPEnrollStartRequest{
			Subject: userID,
			Label:   label,
		})
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("TOTP enroll start failed (check Herald and herald-totp)")
			return SendErrorResponse(ctx, fiber.StatusBadGateway, "TOTP enroll start failed")
		}
		sess.Set(totpEnrollmentIDKey, startResp.EnrollID)
		sess.Set(totpEnrollmentSubjectKey, userID)
		sess.Set(totpEnrollmentStartedKey, time.Now().Unix())
		if err := saveLiveSession(ctx.Context(), store, sess); err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}
		ctx.Set("Cache-Control", "no-store")
		uiWords, uiJSON := authenticationUIData(ctx)
		return ctx.Render("totp_enroll", fiber.Map{
			"Title":     config.LoginPageTitle.Value,
			"PageTitle": uiWords["bind"], "UI": uiWords,
			"UIMessages":        uiJSON,
			"UILanguage":        authenticationUILanguage(ctx),
			"SupportURL":        config.LoginSupportURL.String(),
			"FooterText":        config.LoginPageFooterText.Value,
			"EnrollID":          startResp.EnrollID,
			"OtpauthURI":        template.URL(startResp.OtpauthURI), // avoid html/template sanitizing otpauth:// to #ZgotmplZ
			"HeraldTOTPEnabled": config.HeraldTOTPEnabled.ToBool(),
		})
	}
}

// TOTPEnrollConfirmAPI handles POST /totp/enroll/confirm - confirms TOTP with code (requires auth).
func TOTPEnrollConfirmAPI(store *session.Store) func(c fiber.Ctx) error {
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
		if !reauthenticatedRecently(sess) {
			userID, _ := sess.Get("user_id").(string)
			auditlog.LogTOTPChange(ctx.Context(), userID, auditlog.TOTPActionEnrolled, ctx.IP(), false, "reauthentication_required")
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"ok": false, "error": "recent_authentication_required"})
		}
		enrollID := ctx.FormValue("enroll_id")
		code := ctx.FormValue("code")
		if enrollID == "" || code == "" {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"ok": false, "error": "enroll_id and code required"})
		}
		if invalidTOTPEnrollment(sess, enrollID) {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"ok": false, "error": "invalid_enrollment"})
		}
		boundSubject, _ := sess.Get(totpEnrollmentSubjectKey).(string)
		client := getHeraldClient()
		if client == nil {
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"ok": false, "error": "TOTP service unavailable"})
		}
		confirmResp, err := client.TOTPEnrollConfirm(ctx.Context(), &herald.TOTPEnrollConfirmRequest{
			EnrollID: enrollID,
			Code:     code,
		})
		if err != nil {
			log.Warn().Err(err).Str("enroll_id", enrollID).Msg("TOTP enroll confirm failed")
			var providerError *herald.HeraldError
			if errors.As(err, &providerError) && providerError.StatusCode == fiber.StatusTooManyRequests {
				return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"ok": false, "error": "rate_limited"})
			}
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"ok": false, "error": "invalid_code"})
		}
		if confirmResp == nil {
			log.Error().Str("enroll_id", enrollID).Msg("TOTP enroll confirm returned an empty response")
			return ctx.Status(fiber.StatusBadGateway).JSON(fiber.Map{"ok": false, "error": "invalid_provider_response"})
		}
		if confirmResp.Subject != boundSubject || !confirmResp.TotpEnabled {
			log.Error().Str("expected_subject", boundSubject).Str("confirmed_subject", confirmResp.Subject).Msg("TOTP enrollment subject mismatch")
			return ctx.Status(fiber.StatusBadGateway).JSON(fiber.Map{"ok": false, "error": "enrollment_subject_mismatch"})
		}
		auditlog.LogTOTPChange(ctx.Context(), boundSubject, auditlog.TOTPActionEnrolled, ctx.IP(), true, "")
		clearTOTPEnrollment(sess)
		if err := sess.Save(); err != nil {
			// Herald has already completed enrollment and backup codes are only
			// returned once. Preserve that authoritative result for the user;
			// stale local enrollment metadata expires after a short interval.
			log.Error().Err(err).Str("enroll_id", enrollID).Msg("Failed to clear completed TOTP enrollment from session")
		}
		return ctx.JSON(fiber.Map{
			"ok":           true,
			"subject":      confirmResp.Subject,
			"totp_enabled": confirmResp.TotpEnabled,
			"backup_codes": confirmResp.BackupCodes,
		})
	}
}

func invalidTOTPEnrollment(sess *session.Session, enrollID string) bool {
	boundEnrollID, _ := sess.Get(totpEnrollmentIDKey).(string)
	boundSubject, _ := sess.Get(totpEnrollmentSubjectKey).(string)
	startedAt, startedOK := sess.Get(totpEnrollmentStartedKey).(int64)
	userID, _ := sess.Get("user_id").(string)
	enrollmentAge := time.Since(time.Unix(startedAt, 0))
	return boundEnrollID == "" || boundSubject == "" || userID == "" ||
		enrollID != boundEnrollID || boundSubject != userID || !startedOK ||
		enrollmentAge < 0 || enrollmentAge > totpEnrollmentValidity
}
