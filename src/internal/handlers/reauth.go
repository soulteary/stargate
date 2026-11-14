package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auditlog"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/warden/pkg/warden"
)

const (
	// reauthValidity is how long a proof given to start TOTP enrollment
	// authorizes that enrollment.
	reauthValidity = 5 * time.Minute
	// reauthVerifiedAtKey holds the time of a proof given to POST /totp/enroll.
	// Only that request writes it: a login, a step-up and a TOTP revocation do
	// not. The key is not the "reauth_verified_at" of earlier releases, which
	// any proof set, so a record written before an upgrade is not accepted.
	reauthVerifiedAtKey = "totp_enroll_proof_at"
	// reauthUsedAtKey marks the proof as spent: an enrollment was started
	// with it. A spent proof still covers the confirmation of that enrollment
	// but starts no other.
	reauthUsedAtKey = "totp_enroll_proof_used_at"
	// reauthChallengeIDKey binds the delivered code to the session it was
	// requested from.
	reauthChallengeIDKey = "reauth_challenge_id"

	// reauthChallengePurpose marks the gateway-side challenge context so that a
	// login code cannot be replayed as a re-authentication code or vice versa.
	reauthChallengePurpose = "reauth"
	// reauthHeraldPurpose is the purpose sent to Herald. Herald only accepts
	// purposes listed in its ALLOWED_PURPOSES setting, which defaults to
	// "login"; a dedicated purpose would fail on a default Herald deployment.
	reauthHeraldPurpose = "login"
)

// Proofs a session may be asked for before a sensitive operation.
const (
	reauthProofNone     = ""
	reauthProofTOTP     = "totp"
	reauthProofCode     = "code"
	reauthProofPassword = "password"
)

var errReauthProviderUnavailable = errors.New("re-authentication provider is unavailable")

// reauthFailure describes why a sensitive operation may not proceed.
type reauthFailure struct {
	status  int
	code    string
	message string
}

var (
	reauthFailed          = &reauthFailure{fiber.StatusUnauthorized, "recent_authentication_required", "additional authentication failed"}
	reauthNotAvailable    = &reauthFailure{fiber.StatusForbidden, "reauthentication_unavailable", "re-authentication is not available for this session"}
	reauthProviderFailure = &reauthFailure{fiber.StatusBadGateway, "service_unavailable", "re-authentication service is unavailable"}
)

// reauthProofFor selects the single proof the session may present. The choice
// is made from server-side state only, so a client cannot downgrade it:
//
//   - a user with TOTP enrolled must present a current TOTP code;
//   - any other Warden user must present a code delivered to the contact
//     registered in Warden;
//   - only a session created by the shared password, which carries no user
//     identity, may present that password again.
//
// A session with a user identity never falls back to the shared password.
func reauthProofFor(ctx context.Context, sess *session.Session) (string, error) {
	userID, _ := sess.Get("user_id").(string)
	directory := wardenSessionUsesDirectory(sess)
	if userID == "" && !directory {
		if config.Passwords.Value == "" {
			return reauthProofNone, nil
		}
		return reauthProofPassword, nil
	}
	client := getHeraldClient()
	if client == nil || userID == "" {
		return reauthProofNone, nil
	}
	if config.HeraldTOTPEnabled.ToBool() {
		status, err := client.TOTPStatus(ctx, userID)
		if err != nil || status == nil {
			return reauthProofNone, errReauthProviderUnavailable
		}
		if status.TotpEnabled {
			return reauthProofTOTP, nil
		}
	}
	if directory {
		return reauthProofCode, nil
	}
	return reauthProofNone, nil
}

// requiredReauthProof resolves the proof for a page or an operation and maps
// the cases in which none can be offered to a response.
func requiredReauthProof(ctx context.Context, sess *session.Session) (string, *reauthFailure) {
	proof, err := reauthProofFor(ctx, sess)
	if err != nil {
		return reauthProofNone, reauthProviderFailure
	}
	if proof == reauthProofNone {
		return reauthProofNone, reauthNotAvailable
	}
	return proof, nil
}

// reauthenticate verifies the proof submitted with the request: form field
// "password" for the shared password, "code" for a TOTP or delivered code.
// It records nothing about the result: a proof counts for the request that
// carries it, and the caller decides what that request may record. The caller
// must persist sess with saveLiveSession, because a delivered code that was
// verified is removed from it.
func reauthenticate(ctx fiber.Ctx, sess *session.Session) *reauthFailure {
	proof, failure := requiredReauthProof(ctx.Context(), sess)
	if failure != nil {
		return failure
	}
	userID, _ := sess.Get("user_id").(string)
	code := strings.TrimSpace(ctx.FormValue("code"))
	verified := false
	switch proof {
	case reauthProofPassword:
		verified = auth.CheckPassword(ctx.FormValue("password"))
	case reauthProofTOTP:
		if code != "" {
			resp, err := getHeraldClient().TOTPVerify(ctx.Context(), &herald.TOTPVerifyRequest{Subject: userID, Code: code})
			verified = err == nil && resp != nil && resp.OK
		}
	case reauthProofCode:
		verified = verifyReauthCode(ctx, sess, userID, code)
	}
	if !verified {
		return reauthFailed
	}
	return nil
}

// recordEnrollmentProof stores a proof that was given to start TOTP
// enrollment and has not started one yet.
func recordEnrollmentProof(sess *session.Session) {
	sess.Set(reauthVerifiedAtKey, time.Now().Unix())
	sess.Delete(reauthUsedAtKey)
}

// spendEnrollmentProof marks the stored proof as used by an enrollment start.
func spendEnrollmentProof(sess *session.Session) {
	sess.Set(reauthUsedAtKey, time.Now().Unix())
}

// discardEnrollmentProof removes the stored proof.
func discardEnrollmentProof(sess *session.Session) {
	sess.Delete(reauthVerifiedAtKey)
	sess.Delete(reauthUsedAtKey)
}

// enrollmentProofUnspent reports whether the session holds a proof that was
// given to start TOTP enrollment, is at most reauthValidity old and has not
// started an enrollment. Such a proof exists only between a start request
// whose proof was accepted and whose call to Herald failed, and the retry.
func enrollmentProofUnspent(sess *session.Session) bool {
	return sess.Get(reauthUsedAtKey) == nil && reauthenticatedRecently(sess)
}

// reauthenticatedRecently reports whether the session holds an enrollment
// proof, spent or not, that is at most reauthValidity old.
func reauthenticatedRecently(sess *session.Session) bool {
	verifiedAt, ok := sess.Get(reauthVerifiedAtKey).(int64)
	age := time.Since(time.Unix(verifiedAt, 0))
	return ok && age >= 0 && age <= reauthValidity
}

// reauthDirectoryUser returns the Warden record of the session's user, or nil
// when the record is unavailable or belongs to a different user.
func reauthDirectoryUser(ctx context.Context, sess *session.Session) *warden.AllowListUser {
	userID, _ := sess.Get("user_id").(string)
	phone, _ := sess.Get("user_phone").(string)
	mail, _ := sess.Get("user_mail").(string)
	user, _, err := lookupRefreshUser(ctx, phone, mail)
	if err != nil || user == nil || userID == "" || user.UserID != userID {
		return nil
	}
	return user
}

// verifyReauthCode checks a delivered code against the challenge this session
// requested. The challenge must be the one stored in the session, must have
// been created for re-authentication, and must still match the user's
// registered contact. Herald then verifies it bound to user, purpose and
// channel, and consumes it.
func verifyReauthCode(ctx fiber.Ctx, sess *session.Session, userID, code string) bool {
	challengeID, _ := sess.Get(reauthChallengeIDKey).(string)
	if challengeID == "" || code == "" {
		return false
	}
	binding, err := loadChallengeContext(ctx.Context(), challengeID)
	if err != nil || binding.Purpose != reauthChallengePurpose {
		return false
	}
	identity := &loginIdentity{userID: userID, wardenUserInfo: reauthDirectoryUser(ctx.Context(), sess)}
	if !identity.matchesLoginChallenge(challengeID, binding) {
		return false
	}
	resp, err := getHeraldClient().VerifyChallengeV2(ctx.Context(), &herald.VerifyChallengeV2Request{
		ChallengeID:     challengeID,
		Code:            code,
		ClientIP:        ctx.IP(),
		ExpectedUserID:  userID,
		ExpectedPurpose: reauthHeraldPurpose,
		ExpectedChannel: binding.Channel,
	})
	if err != nil || resp == nil || !resp.OK || resp.ChallengeID != challengeID || resp.UserID != userID ||
		resp.Purpose != reauthHeraldPurpose || resp.Channel != binding.Channel {
		reason := "invalid"
		if resp != nil && !resp.OK && resp.Reason != "" {
			reason = resp.Reason
		} else if err != nil {
			reason = "provider_error"
		}
		auditlog.LogReauthCodeCheck(ctx.Context(), userID, challengeID, binding.Channel, binding.Destination, ctx.IP(), false, reason)
		return false
	}
	auditlog.LogReauthCodeCheck(ctx.Context(), userID, challengeID, binding.Channel, binding.Destination, ctx.IP(), true, "")
	if err := getChallengeContextStore().Delete(ctx.Context(), challengeID); err != nil {
		log.Error().Err(err).Str("challenge_id", challengeID).Msg("Failed to consume re-authentication challenge context")
	}
	sess.Delete(reauthChallengeIDKey)
	return true
}

// reauthView adds the fields the proof form needs to a page's view data.
func reauthView(view fiber.Map, proof string) fiber.Map {
	view["Reauth"] = proof
	view["LoginSMSEnabled"] = config.LoginSMSEnabled.ToBool()
	view["LoginEmailEnabled"] = config.LoginEmailEnabled.ToBool()
	return view
}

// ReauthSendCodeAPI handles POST /_send_reauth_code. It sends a one-time code
// to the contact registered in Warden for the session's user. The request can
// choose a delivery channel but never a destination.
func ReauthSendCodeAPI(store *session.Store) func(c fiber.Ctx) error {
	return func(ctx fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"), "session_store_failed")
		}
		defer sess.Release()
		if !auth.IsAuthenticated(sess) {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusUnauthorized, "authentication required", "unauthorized")
		}
		if allowed, err := authorizeSensitiveSession(ctx, store, sess); !allowed {
			return err
		}
		proof, failure := requiredReauthProof(ctx.Context(), sess)
		if failure != nil {
			return sendVerifyCodeErrorJSON(ctx, failure.status, failure.message, failure.code)
		}
		if proof != reauthProofCode {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, "a delivered code is not the proof for this session", "reauthentication_unavailable")
		}
		userID, _ := sess.Get("user_id").(string)
		deliverVia := ctx.FormValue("deliver_via")
		channel, destination, selectionReason := selectVerificationDestination(reauthDirectoryUser(ctx.Context(), sess), deliverVia)
		if selectionReason != "" {
			return verificationSelectionError(ctx, deliverVia, selectionReason)
		}

		createResp, metadata, err := getHeraldClient().CreateChallengeWithMetadata(ctx.Context(), &herald.CreateChallengeRequest{
			UserID:      userID,
			Channel:     channel,
			Destination: destination,
			Purpose:     reauthHeraldPurpose,
			Locale:      verificationLocale(ctx),
			ClientIP:    ctx.IP(),
			UA:          ctx.Get("User-Agent"),
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to create re-authentication challenge")
			return handleChallengeCreationErrorWithMetadata(ctx, err, metadata, userID, channel, destination)
		}
		auditlog.LogReauthCodeSend(ctx.Context(), userID, createResp.ChallengeID, channel, destination, ctx.IP(), true, "")
		binding := challengeContext{
			ChallengeID: createResp.ChallengeID,
			UserID:      userID,
			Channel:     channel,
			Destination: destination,
			Purpose:     reauthChallengePurpose,
		}
		if err := bindChallengeContext(ctx.Context(), binding, createResp.ExpiresIn); err != nil {
			log.Error().Err(err).Str("challenge_id", createResp.ChallengeID).Msg("Failed to persist re-authentication challenge context")
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"), "binding_unavailable")
		}
		sess.Set(reauthChallengeIDKey, createResp.ChallengeID)
		if err := saveLiveSession(ctx.Context(), store, sess); err != nil {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"), "session_store_failed")
		}
		return sendChallengeCreated(ctx, createResp)
	}
}
