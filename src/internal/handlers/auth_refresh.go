package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
)

var lookupRefreshUser = func(ctx context.Context, phone, mail string) *warden.AllowListUser {
	return auth.GetUserInfo(ctx, phone, mail)
}

func refreshAuthorizationIfNeeded(ctx context.Context, sess *session.Session) (bool, error) {
	// Authorization refresh only applies to authenticated Warden sessions. In
	// particular, an anonymous request must continue through the normal
	// ForwardAuth flow so that HTML clients are redirected to login.
	if !config.AuthRefreshEnabled.ToBool() || !config.WardenEnabled.ToBool() || !auth.IsAuthenticated(sess) {
		return false, nil
	}
	method, _ := sess.Get("auth_method").(string)
	phone, _ := sess.Get("user_phone").(string)
	mail, _ := sess.Get("user_mail").(string)
	if method != "warden" {
		// Sessions created before auth_method was recorded can still be identified
		// by their Warden identity. Explicit non-Warden sessions must never be sent
		// through the global Warden refresh path.
		if method != "" || (phone == "" && mail == "") {
			return false, nil
		}
	}
	interval := config.AuthRefreshInterval.ToDuration()
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	if refreshedAt, ok := sess.Get("auth_refreshed_at").(int64); ok &&
		time.Since(time.Unix(refreshedAt, 0)) <= interval {
		return false, nil
	}
	if phone == "" && mail == "" {
		return false, errors.New("session has no refresh identity")
	}
	user := lookupRefreshUser(ctx, phone, mail)
	if user == nil || (user.Status != "" && !strings.EqualFold(user.Status, "active")) {
		_ = auth.Unauthenticate(sess)
		return false, errors.New("user is no longer active")
	}
	sess.Set("user_id", user.UserID)
	sess.Set("user_mail", user.Mail)
	sess.Set("user_phone", user.Phone)
	sess.Set("user_scope", user.Scope)
	sess.Set("user_role", user.Role)
	sess.Set("user_name", user.Name)
	sess.Set("user_status", user.Status)
	sess.Set("auth_refreshed_at", time.Now().Unix())
	return true, nil
}
