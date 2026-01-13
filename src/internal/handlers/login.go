package handlers

import (
	"fmt"
	"html"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// LoginAPI handles POST requests to /_login for password authentication.
// It validates the password from the form data, creates a session if valid,
// and redirects to the callback URL (if provided) or returns a success response.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginAPI(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		password := ctx.FormValue("password")

		if !auth.CheckPassword(password) {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
		}

		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		err = auth.Authenticate(sess)
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
					sess, err = store.Get(ctx)
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
	return func(ctx *fiber.Ctx) error {
		// Get callback parameter (priority: URL query parameter, then cookie)
		// URL parameter takes priority as it represents the explicit intent of the current request
		callback := ctx.Query("callback")
		if callback == "" {
			callback = GetCallbackFromCookie(ctx)
		} else {
			// If URL has callback parameter, update cookie (if domain is different)
			SetCallbackCookie(ctx, callback)
		}

		sess, err := store.Get(ctx)
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

		return ctx.Render("login", fiber.Map{
			"Callback":   callback,
			"SessionID":  sess.ID(),
			"Title":      config.LoginPageTitle.Value,
			"FooterText": config.LoginPageFooterText.Value,
		})
	}
}
