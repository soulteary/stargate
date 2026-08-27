package handlers

import (
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

const stepUpValidity = 10 * time.Minute

func stepUpVerified(sess *session.Session) bool {
	verifiedAt, ok := sess.Get("step_up_verified_at").(int64)
	age := time.Since(time.Unix(verifiedAt, 0))
	return ok && age >= 0 && age <= stepUpValidity
}

func requiresStepUp(ctx fiber.Ctx, sess *session.Session) bool {
	if !config.StepUpEnabled.ToBool() || !auth.IsAuthenticated(sess) || stepUpVerified(sess) {
		return false
	}
	raw, ok := stepUpForwardedURI(ctx)
	if !ok {
		return true
	}
	path, ok := stepUpRequestPath(raw)
	if !ok {
		return true
	}
	return config.GetStepUpMatcher().RequiresStepUp(path)
}

// stepUpForwardedURI returns the original request URI only when it came from a
// trusted proxy. Falling back to Stargate's own /_auth path would silently
// disable every configured business-path rule.
func stepUpForwardedURI(ctx fiber.Ctx) (string, bool) {
	if !trustForwardedHeaders(ctx) {
		return "", false
	}
	forwardedURI := strings.TrimSpace(ctx.Get("X-Forwarded-Uri"))
	return forwardedURI, forwardedURI != ""
}

// stepUpRequestPath removes the query and fragment before matching protected
// routes. ForwardAuth proxies commonly send X-Forwarded-Uri as a request URI;
// matching the raw value lets `/admin?x=1` bypass an exact `/admin` rule.
func stepUpRequestPath(raw string) (string, bool) {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") || strings.Contains(parsed.Path, "\\") || hasDotPathSegment(parsed.Path) {
		return "", false
	}
	return parsed.Path, true
}

func safeStepUpCallback(raw string) string {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") || strings.Contains(parsed.Path, "\\") || hasDotPathSegment(parsed.Path) || parsed.Fragment != "" {
		return "/"
	}
	return raw
}

func hasDotPathSegment(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func redirectToStepUp(ctx fiber.Ctx) error {
	forwardedURI, _ := stepUpForwardedURI(ctx)
	callback := safeStepUpCallback(forwardedURI)
	callbackHost, err := ValidateCallbackHost(GetForwardedHost(ctx))
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusForbidden, "step-up callback host is not allowed")
	}
	stepUpURL := url.URL{Scheme: GetForwardedProto(ctx), Host: config.AuthHost.String(), Path: "/_step_up"}
	query := stepUpURL.Query()
	query.Set("callback", callback)
	query.Set("callback_host", callbackHost)
	query.Set("callback_proto", GetForwardedProto(ctx))
	stepUpURL.RawQuery = query.Encode()
	return ctx.Redirect().Status(fiber.StatusFound).To(stepUpURL.String())
}

func stepUpReturnURL(rawProto, rawHost, rawCallback string) string {
	callback := safeStepUpCallback(rawCallback)
	host, err := ValidateCallbackHost(rawHost)
	if err != nil {
		return callback
	}
	proto := strings.ToLower(strings.TrimSpace(rawProto))
	if proto != "http" && proto != "https" {
		return callback
	}
	parsed, err := url.ParseRequestURI(callback)
	if err != nil {
		return "/"
	}
	parsed.Scheme = proto
	parsed.Host = host
	return parsed.String()
}

func StepUpRoute(store *session.Store) func(c fiber.Ctx) error {
	return func(ctx fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, "authentication required")
		}
		defer sess.Release()
		if !auth.IsAuthenticated(sess) {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, "authentication required")
		}
		callback := safeStepUpCallback(ctx.Query("callback"))
		callbackHost, _ := ValidateCallbackHost(ctx.Query("callback_host"))
		callbackProto := strings.ToLower(strings.TrimSpace(ctx.Query("callback_proto")))
		if callbackProto != "http" && callbackProto != "https" {
			callbackProto = ""
		}
		return ctx.Type("html").SendString(`<!doctype html><html><body><form method="post" action="/_step_up"><input type="password" name="password" autocomplete="current-password" required><input type="hidden" name="callback" value="` + html.EscapeString(callback) + `"><input type="hidden" name="callback_host" value="` + html.EscapeString(callbackHost) + `"><input type="hidden" name="callback_proto" value="` + html.EscapeString(callbackProto) + `"><button type="submit">Verify</button></form></body></html>`)
	}
}

func StepUpAPI(store *session.Store) func(c fiber.Ctx) error {
	return func(ctx fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, "authentication required")
		}
		defer sess.Release()
		if !auth.IsAuthenticated(sess) {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, "authentication required")
		}
		if !auth.CheckPassword(ctx.FormValue("password")) {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, "additional authentication failed")
		}
		sess.Set("step_up_verified_at", time.Now().Unix())
		if err := sess.Save(); err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, "failed to save additional authentication")
		}
		return ctx.Redirect().Status(fiber.StatusFound).To(stepUpReturnURL(ctx.FormValue("callback_proto"), ctx.FormValue("callback_host"), ctx.FormValue("callback")))
	}
}
