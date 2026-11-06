package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

var lookupRefreshUser = auth.LookupUserInfo

var authorizationNow = time.Now

var errAuthorizationUnavailable = errors.New("directory authorization is unavailable")

func refreshAuthorizationIfNeeded(ctx context.Context, sess *session.Session) (bool, error) {
	return refreshDirectoryAuthorization(ctx, sess, false)
}

func refreshDirectoryAuthorization(ctx context.Context, sess *session.Session, force bool) (bool, error) {
	// Authorization refresh only applies to authenticated Warden sessions. In
	// particular, an anonymous request must continue through the normal
	// ForwardAuth flow so that HTML clients are redirected to login.
	if !config.WardenEnabled.ToBool() || !auth.IsAuthenticated(sess) || !wardenSessionUsesDirectory(sess) {
		return false, nil
	}
	if !force && ((!config.AuthRefreshEnabled.ToBool() && config.WardenSourceMaxAge.ToDuration() <= 0) || !wardenSessionNeedsRefresh(sess)) {
		return false, nil
	}
	phone, _ := sess.Get("user_phone").(string)
	mail, _ := sess.Get("user_mail").(string)

	if phone == "" && mail == "" {
		return false, errors.New("session has no refresh identity")
	}
	user, deadline, lookupErr := lookupRefreshUser(ctx, phone, mail)
	if lookupErr != nil && !errors.Is(lookupErr, auth.ErrUserUnavailable) || user == nil && ctx.Err() != nil {
		// A canceled or expired request does not prove that the Warden user was
		// removed or disabled. Fail the current refresh closed, but preserve the
		// authenticated session so a transient timeout does not log the user out.
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, errAuthorizationUnavailable
	}
	if user == nil || (user.Status != "" && !strings.EqualFold(user.Status, "active")) {
		_ = auth.Unauthenticate(sess)
		return false, errors.New("user is no longer active")
	}
	// Contacts can be reassigned; they locate records but cannot authorize
	// rebinding a session to a different stable directory identity.
	boundUserID, _ := sess.Get("user_id").(string)
	if boundUserID == "" || user.UserID == "" || boundUserID != user.UserID {
		_ = auth.Unauthenticate(sess)
		return false, errors.New("session refresh identity does not match; sign in again")
	}
	sess.Set("user_id", user.UserID)
	// Losing or changing the verified address invalidates the evidence
	// permanently; changing it back later cannot resurrect the old proof.
	oldEmail, oldValid := canonicalEvidenceEmail(mail)
	newEmail, newValid := canonicalEvidenceEmail(user.Mail)
	if !oldValid || !newValid || oldEmail != newEmail {
		clearVerifiedEmail(sess)
	}
	sess.Set("user_mail", user.Mail)
	sess.Set("user_phone", user.Phone)
	sess.Set("user_scope", user.Scope)
	sess.Set("user_role", user.Role)
	sess.Set("user_name", user.Name)
	sess.Set("user_status", user.Status)
	sess.Set("auth_refreshed_at", time.Now().Unix())
	if !deadline.IsZero() {
		sess.Set("warden_source_valid_until", deadline.UnixNano())
		sess.Set("warden_source_policy", wardenSourcePolicy())
	} else {
		sess.Delete("warden_source_valid_until")
		sess.Delete("warden_source_policy")
	}
	return true, nil
}

func wardenSessionUsesDirectory(sess *session.Session) bool {
	method, _ := sess.Get("auth_method").(string)
	phone, _ := sess.Get("user_phone").(string)
	mail, _ := sess.Get("user_mail").(string)
	return method == "warden" || method == "" && (phone != "" || mail != "")
}

func wardenSourcePolicy() string {
	return config.WardenSourceMaxAge.ToDuration().String() + ":" + config.WardenSourceClockSkew.ToDuration().String()
}

func wardenSessionSourceExpired(sess *session.Session) bool {
	if config.WardenSourceMaxAge.ToDuration() <= 0 || !wardenSessionUsesDirectory(sess) {
		return false
	}
	deadline, ok := sess.Get("warden_source_valid_until").(int64)
	return !ok || sess.Get("warden_source_policy") != wardenSourcePolicy() || !authorizationNow().Before(time.Unix(0, deadline))
}

func wardenSessionNeedsRefresh(sess *session.Session) bool {
	if !wardenSessionUsesDirectory(sess) {
		return false
	}
	if wardenSessionSourceExpired(sess) {
		return true
	}
	interval := config.AuthRefreshInterval.ToDuration()
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	if refreshedAt, ok := sess.Get("auth_refreshed_at").(int64); ok &&
		time.Since(time.Unix(refreshedAt, 0)) <= interval {
		return false
	}
	return true
}
