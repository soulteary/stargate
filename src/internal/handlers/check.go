package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// handleNotAuthenticated handles unauthenticated requests.
// For HTML requests, it redirects to the login page.
// For API requests (JSON/XML), it returns a 401 error response.
func handleNotAuthenticated(ctx *fiber.Ctx) error {
	if IsHTMLRequest(ctx) {
		// HTML request: redirect to login page
		callbackURL := BuildCallbackURL(ctx)
		return ctx.Redirect(callbackURL)
	}

	// Non-HTML request: return 401 error
	return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.auth_required"))
}

// CheckRoute is the main authentication check handler for Traefik Forward Auth.
// It validates requests in two ways:
//  1. Stargate-Password header authentication (for API requests)
//  2. Session cookie authentication (for web requests)
//
// On successful authentication, it sets the X-Forwarded-User header (or configured header name)
// and returns 200 OK. On failure, it either redirects to login (HTML) or returns 401 (API).
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func CheckRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			// Session store error, return 500 error
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		// Handle Stargate-Password Header authentication
		stargatePassword := ctx.Get("Stargate-Password")
		if stargatePassword != "" {
			if !auth.CheckPassword(stargatePassword) {
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
			}

			// Authentication successful, set user info header
			// Since Stargate uses password authentication, there's no specific username, use default value
			userHeaderName := config.UserHeaderName.String()
			ctx.Set(userHeaderName, "authenticated")
			return ctx.SendStatus(fiber.StatusOK)
		}

		// Handle Warden user list authentication via headers
		userPhone := ctx.Get("X-User-Phone")
		userMail := ctx.Get("X-User-Mail")
		if userPhone != "" || userMail != "" {
			// Use context from request
			// ctx.Context() returns *fasthttp.RequestCtx which implements context.Context
			// CheckUserInList handles nil context internally by using context.Background()
			if auth.CheckUserInList(ctx.Context(), userPhone, userMail) {
				// Authentication successful, set user info header
				userHeaderName := config.UserHeaderName.String()
				ctx.Set(userHeaderName, "authenticated")
				return ctx.SendStatus(fiber.StatusOK)
			}
			// User not in list, continue to session check
		}

		// Check session authentication
		if !auth.IsAuthenticated(sess) {
			return handleNotAuthenticated(ctx)
		}

		// Authentication successful, set user info headers
		userHeaderName := config.UserHeaderName.String()

		// Get user information from session (for Warden authentication)
		userIDVal := sess.Get("user_id")
		userMailVal := sess.Get("user_mail")
		userScopeVal := sess.Get("user_scope")
		userRoleVal := sess.Get("user_role")

		// Set basic authentication header
		var userID string
		if userIDVal != nil {
			if id, ok := userIDVal.(string); ok {
				userID = id
				ctx.Set(userHeaderName, userID)
			} else {
				ctx.Set(userHeaderName, "authenticated")
			}
		} else {
			// Fallback to default value for password authentication
			ctx.Set(userHeaderName, "authenticated")
		}

		// Set authorization headers for downstream services (as per Claude.md spec)
		if userMailVal != nil {
			if mail, ok := userMailVal.(string); ok && mail != "" {
				ctx.Set("X-Auth-Email", mail)
			}
		}

		if userID != "" {
			ctx.Set("X-Auth-User", userID)
		}

		// Set scope header (comma-separated list)
		if userScopeVal != nil {
			if scopes, ok := userScopeVal.([]string); ok && len(scopes) > 0 {
				ctx.Set("X-Auth-Scopes", strings.Join(scopes, ","))
			} else if scopes, ok := userScopeVal.([]interface{}); ok && len(scopes) > 0 {
				// Handle case where scope is stored as []interface{}
				scopeStrs := make([]string, 0, len(scopes))
				for _, s := range scopes {
					if str, ok := s.(string); ok {
						scopeStrs = append(scopeStrs, str)
					}
				}
				if len(scopeStrs) > 0 {
					ctx.Set("X-Auth-Scopes", strings.Join(scopeStrs, ","))
				}
			}
		}

		// Set role header
		if userRoleVal != nil {
			if role, ok := userRoleVal.(string); ok && role != "" {
				ctx.Set("X-Auth-Role", role)
			}
		}

		return ctx.SendStatus(fiber.StatusOK)
	}
}
