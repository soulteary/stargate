package handlers

import (
	"fmt"
	"html"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/sirupsen/logrus"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// Authenticator defines an interface for authenticating sessions.
// This interface allows for easier testing by enabling mock implementations.
type Authenticator interface {
	Authenticate(sess *session.Session) error
}

// AuthAuthenticator wraps auth.Authenticate to implement Authenticator interface.
type AuthAuthenticator struct{}

// Authenticate marks a session as authenticated.
func (a *AuthAuthenticator) Authenticate(sess *session.Session) error {
	return auth.Authenticate(sess)
}

// loginAPIHandler is the internal handler that can be tested with mocked dependencies.
func loginAPIHandler(ctx *fiber.Ctx, sessionGetter SessionGetter, authenticator Authenticator) error {
	password := ctx.FormValue("password")
	authMethod := ctx.FormValue("auth_method") // "password" or "warden"
	userPhone := ctx.FormValue("phone")
	userMail := ctx.FormValue("mail")

	// Determine authentication method
	// If auth_method is not specified, default to password authentication for backward compatibility
	if authMethod == "" {
		authMethod = "password"
	}

	var authenticated bool

	if authMethod == "warden" {
		// Warden user list authentication
		// Check if at least one identifier is provided
		if userPhone == "" && userMail == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T("error.user_not_in_list"))
		}

		// Log the authentication attempt
		logrus.Debugf("Attempting Warden authentication: phone=%s, mail=%s", userPhone, userMail)

		// Use context from request
		// ctx.Context() returns *fasthttp.RequestCtx which implements context.Context
		// CheckUserInList handles nil context internally by using context.Background()
		if !auth.CheckUserInList(ctx.Context(), userPhone, userMail) {
			logrus.Warnf("Warden authentication failed for: phone=%s, mail=%s", userPhone, userMail)
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.user_not_in_list"))
		}

		logrus.Infof("Warden authentication successful for: phone=%s, mail=%s", userPhone, userMail)
		authenticated = true
	} else {
		// Password authentication (default)
		if password == "" {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
		}
		if !auth.CheckPassword(password) {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
		}
		authenticated = true
	}

	if !authenticated {
		return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.authentication_failed"))
	}

	sess, err := sessionGetter.Get(ctx)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
	}

	// Set user information to session for warden authentication before authenticating
	if authMethod == "warden" {
		if userPhone != "" {
			sess.Set("user_phone", userPhone)
		}
		if userMail != "" {
			sess.Set("user_mail", userMail)
		}
	}

	// Authenticate and save session (this will save all session data including user info)
	err = authenticator.Authenticate(sess)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.authenticate_failed"))
	}

	// Get callback parameter (priority: cookie, form data, query parameter)
	callbackFromCookie := GetCallbackFromCookie(ctx)
	callback := callbackFromCookie
	if callback == "" {
		callback = ctx.FormValue("callback")
	}
	if callback == "" {
		callback = ctx.Query("callback")
	}

	// If callback was retrieved from cookie, clear the cookie after successful login
	if callbackFromCookie != "" {
		ClearCallbackCookie(ctx)
	}

	// If no callback, try using origin host as callback
	if callback == "" {
		originHost := GetForwardedHost(ctx)
		// Only use origin host as callback if it's different from auth service domain
		if IsDifferentDomain(ctx) {
			callback = originHost
		}
	}

	// If callback exists, redirect to session exchange endpoint
	if callback != "" {
		// Get session ID (should already exist)
		sessionID := sess.ID()
		if sessionID == "" {
			// If ID is empty, try to get it from response cookie
			// In Fiber session, Save() will set session ID to response cookie
			cookieBytes := ctx.Response().Header.Peek("Set-Cookie")
			if len(cookieBytes) > 0 {
				cookieStr := string(cookieBytes)
				// Find session cookie (format: stargate_session=<session_id>; ...)
				cookieName := auth.SessionCookieName + "="
				if idx := strings.Index(cookieStr, cookieName); idx >= 0 {
					start := idx + len(cookieName)
					end := start
					for end < len(cookieStr) && cookieStr[end] != ';' && cookieStr[end] != ' ' {
						end++
					}
					sessionID = cookieStr[start:end]
				}
			}
			// If still empty, try to get session again
			if sessionID == "" {
				sess, err = sessionGetter.Get(ctx)
				if err != nil {
					return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
				}
				sessionID = sess.ID()
			}
			if sessionID == "" {
				// If session ID is still empty, return error
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.missing_session_id"))
			}
		}
		proto := GetForwardedProto(ctx)
		if proto == "" {
			proto = ctx.Protocol()
		}
		redirectURL := fmt.Sprintf("%s://%s/_session_exchange?id=%s", proto, callback, sessionID)
		return ctx.Redirect(redirectURL)
	}

	// If still no callback (origin domain is the auth service itself), return response based on request type
	if IsHTMLRequest(ctx) {
		// HTML request returns success message and adds meta refresh redirect to origin domain
		ctx.Set("Content-Type", "text/html; charset=utf-8")
		successMsg := i18n.T("success.login")

		// Get origin host and protocol
		originHost := GetForwardedHost(ctx)
		proto := GetForwardedProto(ctx)
		redirectURL := fmt.Sprintf("%s://%s", proto, originHost)

		// Escape URL to ensure HTML safety
		escapedURL := html.EscapeString(redirectURL)

		// Build HTML with meta refresh
		htmlContent := fmt.Sprintf(`<html><head><meta charset="UTF-8"><meta http-equiv="refresh" content="0;url=%s"><title>%s</title></head><body><h1>%s</h1><p>%s</p><p><a href="%s">点击这里如果页面没有自动跳转</a></p></body></html>`,
			escapedURL, successMsg, successMsg, successMsg, escapedURL)
		return ctx.Status(fiber.StatusOK).SendString(htmlContent)
	}

	// API request returns JSON response
	ctx.Set("Content-Type", "application/json")
	response := fiber.Map{
		"success": true,
		"message": i18n.T("success.login"),
	}
	// If session ID exists, add it to response
	if sessionID := sess.ID(); sessionID != "" {
		response["session_id"] = sessionID
	}
	return ctx.Status(fiber.StatusOK).JSON(response)
}

// LoginAPI handles POST requests to /_login for password authentication.
// It validates the password from the form data, creates a session if valid,
// and redirects to the callback URL (if provided) or returns a success response.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginAPI(store *session.Store) func(c *fiber.Ctx) error {
	sessionGetter := &SessionStoreAdapter{store: store}
	authenticator := &AuthAuthenticator{}
	return func(ctx *fiber.Ctx) error {
		return loginAPIHandler(ctx, sessionGetter, authenticator)
	}
}

// loginRouteHandler is the internal handler that can be tested with mocked dependencies.
func loginRouteHandler(ctx *fiber.Ctx, sessionGetter SessionGetter) error {
	// Get callback parameter (priority: URL query parameter, then cookie)
	// URL parameter takes priority as it represents the explicit intent of the current request
	callback := ctx.Query("callback")
	if callback == "" {
		callback = GetCallbackFromCookie(ctx)
	} else {
		// If URL has callback parameter, update cookie (if domain is different)
		SetCallbackCookie(ctx, callback)
	}

	sess, err := sessionGetter.Get(ctx)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
	}

	if auth.IsAuthenticated(sess) {
		// Use X-Forwarded-* headers to build correct redirect URL
		sessionID := sess.ID()
		if sessionID == "" {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.missing_session_id"))
		}
		proto := GetForwardedProto(ctx)
		if proto == "" {
			proto = ctx.Protocol()
		}
		// If callback exists, redirect to callback's _session_exchange endpoint
		// If no callback, redirect to current host's root path
		if callback != "" {
			redirectURL := fmt.Sprintf("%s://%s/_session_exchange?id=%s", proto, callback, sessionID)
			return ctx.Redirect(redirectURL)
		}
		// When no callback, redirect to current host's root path
		host := GetForwardedHost(ctx)
		redirectURL := fmt.Sprintf("%s://%s/", proto, host)
		return ctx.Redirect(redirectURL)
	}

	// Select template based on Warden configuration
	templateName := "login"
	if config.WardenEnabled.ToBool() {
		templateName = "login.warden"
	}

	return ctx.Render(templateName, fiber.Map{
		"Callback":      callback,
		"SessionID":     sess.ID(),
		"Title":         config.LoginPageTitle.Value,
		"FooterText":    config.LoginPageFooterText.Value,
		"WardenEnabled": config.WardenEnabled.ToBool(),
	})
}

// LoginRoute handles GET requests to /_login for displaying the login page.
// If the user is already authenticated, it redirects to the session exchange endpoint.
// Otherwise, it renders the login page template.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginRoute(store *session.Store) func(c *fiber.Ctx) error {
	sessionGetter := &SessionStoreAdapter{store: store}
	return func(ctx *fiber.Ctx) error {
		return loginRouteHandler(ctx, sessionGetter)
	}
}
