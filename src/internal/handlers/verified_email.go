package handlers

import (
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	forwardauth "github.com/soulteary/forwardauth-kit/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"golang.org/x/net/idna"
)

const verifiedEmailHeader = "X-Auth-Email-Verified"

// Use primitive session values so both memory and Redis stores can preserve the
// evidence without introducing a new serialization contract. AMR is not proof.
var verifiedEmailKeys = [...]string{
	"verified_email_subject", "verified_email_address", "verified_email_at",
	"verified_email_source", "verified_email_version",
}

func clearVerifiedEmail(sess *session.Session) {
	for _, key := range verifiedEmailKeys {
		sess.Delete(key)
	}
}

// canonicalEvidenceEmail deliberately preserves the local part and plus tags.
// Unsupported addresses can still use existing login paths, but cannot acquire
// this declaration. No provider-specific alias or case folding is inferred.
func canonicalEvidenceEmail(value string) (string, bool) {
	if value == "" || len(value) > 254 || strings.TrimSpace(value) != value {
		return "", false
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Name != "" || parsed.Address != value {
		return "", false
	}
	at := strings.LastIndexByte(value, '@')
	if at < 1 || at == len(value)-1 {
		return "", false
	}
	local, domain := value[:at], value[at+1:]
	if len(local) > 64 || strings.ContainsAny(local, "\"\\ ()<>[]:;,") {
		return "", false
	}
	for _, c := range local {
		if c > unicode.MaxASCII || unicode.IsControl(c) {
			return "", false
		}
	}
	ascii, err := idna.Lookup.ToASCII(domain)
	if err != nil || len(ascii) > 253 {
		return "", false
	}
	for _, label := range strings.Split(ascii, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", false
		}
		for _, c := range label {
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' {
				return "", false
			}
		}
	}
	return local + "@" + strings.ToLower(ascii), true
}

// recordVerifiedEmail is called only after Herald accepted the challenge and
// its subject matched the login subject. The server-side challenge destination
// must additionally match the explicit Warden identity's current email.
func (identity *loginIdentity) recordVerifiedEmail(challengeID string, evidence challengeContext, now time.Time) {
	identity.verifiedEmail, identity.verifiedAt = "", 0
	if identity.wardenUserInfo == nil || identity.wardenUserInfo.UserID == "" ||
		identity.userID != identity.wardenUserInfo.UserID ||
		challengeID == "" || evidence.ChallengeID != challengeID ||
		evidence.UserID != identity.userID || evidence.Channel != "email" {
		return
	}
	destination, valid := canonicalEvidenceEmail(evidence.Destination)
	current, currentValid := canonicalEvidenceEmail(identity.wardenUserInfo.Mail)
	if !valid || !currentValid || destination != current {
		return
	}
	identity.verifiedEmail, identity.verifiedAt = current, now.Unix()
}

func storeVerifiedEmail(sess *session.Session, identity *loginIdentity) {
	subject, _ := sess.Get("user_id").(string)
	address, _ := sess.Get("user_mail").(string)
	canonical, valid := canonicalEvidenceEmail(address)
	if !valid || identity.verifiedAt <= 0 || identity.verifiedEmail != canonical || subject == "" || subject != identity.userID {
		return
	}
	sess.Set("verified_email_subject", subject)
	sess.Set("verified_email_address", canonical)
	sess.Set("verified_email_at", identity.verifiedAt)
	sess.Set("verified_email_source", "herald-email-challenge")
	sess.Set("verified_email_version", int64(1))
}

func setVerifiedEmailHeader(ctx fiber.Ctx, sess *session.Session, result *forwardauth.AuthResult, now time.Time) {
	ctx.Set(verifiedEmailHeader, "false")
	if result == nil || !result.Authenticated || result.AuthMethod != forwardauth.AuthMethodSession {
		return
	}
	method, _ := sess.Get("auth_method").(string)
	subject, _ := sess.Get("verified_email_subject").(string)
	address, _ := sess.Get("verified_email_address").(string)
	at, _ := sess.Get("verified_email_at").(int64)
	source, _ := sess.Get("verified_email_source").(string)
	version, _ := sess.Get("verified_email_version").(int64)
	current, valid := canonicalEvidenceEmail(result.Email)
	// Evidence expires independently of sliding session access and is never
	// renewed by directory refresh. Future timestamps fail closed as well.
	if method == "warden" && version == 1 && source == "herald-email-challenge" &&
		at > 0 && at <= now.Unix() && now.Sub(time.Unix(at, 0)) < config.SessionExpiration &&
		subject != "" && subject == result.UserID && valid && address == current {
		ctx.Set(verifiedEmailHeader, "true")
	}
}
