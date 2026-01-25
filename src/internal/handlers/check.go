package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/sirupsen/logrus"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/metrics"
	"github.com/soulteary/tracing-kit"
	"go.opentelemetry.io/otel/attribute"
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
	return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.auth_required"))
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
		// Get trace context from middleware
		traceCtx := ctx.Locals("trace_context")
		if traceCtx == nil {
			traceCtx = ctx.Context()
		}
		spanCtx := traceCtx.(context.Context)

		// Start span for forward auth check
		forwardAuthCtx, forwardAuthSpan := tracing.StartSpan(spanCtx, "auth.forward_auth")
		defer forwardAuthSpan.End()

		forwardAuthSpan.SetAttributes(
			attribute.String("http.path", ctx.Path()),
			attribute.String("http.method", ctx.Method()),
		)

		sess, err := store.Get(ctx)
		if err != nil {
			tracing.RecordError(forwardAuthSpan, err)
			// Session store error, return 500 error
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}

		// Handle Stargate-Password Header authentication
		stargatePassword := ctx.Get("Stargate-Password")
		if stargatePassword != "" {
			if !auth.CheckPassword(stargatePassword) {
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.invalid_password"))
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
			// Start span for Warden check
			wardenCtx, wardenSpan := tracing.StartSpan(forwardAuthCtx, "warden.check_user")
			wardenSpan.SetAttributes(
				attribute.String("warden.identifier_type", func() string {
					if userPhone != "" {
						return "phone"
					}
					return "mail"
				}()),
			)
			// Use context from request
			// ctx.Context() returns *fasthttp.RequestCtx which implements context.Context
			// CheckUserInList handles nil context internally by using context.Background()
			if auth.CheckUserInList(wardenCtx, userPhone, userMail) {
				wardenSpan.SetAttributes(attribute.Bool("warden.user_found", true))
				wardenSpan.End()
				// Authentication successful, set user info header
				userHeaderName := config.UserHeaderName.String()
				ctx.Set(userHeaderName, "authenticated")
				forwardAuthSpan.SetAttributes(attribute.Bool("auth.authenticated", true))
				return ctx.SendStatus(fiber.StatusOK)
			}
			wardenSpan.SetAttributes(attribute.Bool("warden.user_found", false))
			wardenSpan.End()
			// User not in list, continue to session check
		}

		// Check session authentication
		if !auth.IsAuthenticated(sess) {
			forwardAuthSpan.SetAttributes(attribute.Bool("auth.authenticated", false))
			return handleNotAuthenticated(ctx)
		}

		// Check step-up authentication for sensitive paths
		stepUpMatcher := config.GetStepUpMatcher()
		if stepUpMatcher.RequiresStepUp(ctx.Path()) {
			// Check if step-up authentication has been completed
			stepUpVerified := sess.Get("step_up_verified")
			if stepUpVerified == nil || !stepUpVerified.(bool) {
				// Step-up authentication required but not completed
				// Return 403 Forbidden to indicate additional authentication is needed
				if IsHTMLRequest(ctx) {
					// For HTML requests, redirect to step-up verification page
					callbackURL := BuildCallbackURL(ctx)
					stepUpURL := "/_step_up?callback=" + callbackURL
					return ctx.Redirect(stepUpURL)
				}
				// For API requests, return 403
				return SendErrorResponse(ctx, fiber.StatusForbidden, i18n.T(ctx, "error.step_up_required"))
			}
		}

		// Check if auth refresh is enabled and needed
		if config.AuthRefreshEnabled.ToBool() && config.WardenEnabled.ToBool() {
			lastRefreshVal := sess.Get("auth_refreshed_at")
			refreshInterval := config.AuthRefreshInterval.ToDuration()
			if refreshInterval == 0 {
				refreshInterval = 5 * time.Minute // Default 5 minutes
			}

			needsRefresh := false
			if lastRefreshVal == nil {
				needsRefresh = true
			} else if lastRefreshTime, ok := lastRefreshVal.(int64); ok {
				lastRefresh := time.Unix(lastRefreshTime, 0)
				if time.Since(lastRefresh) > refreshInterval {
					needsRefresh = true
				}
			} else {
				// Invalid type, refresh anyway
				needsRefresh = true
			}

			if needsRefresh {
				// Get user identifiers from session
				userPhoneVal := sess.Get("user_phone")
				userMailVal := sess.Get("user_mail")
				var userPhone, userMail string
				if userPhoneVal != nil {
					if phone, ok := userPhoneVal.(string); ok {
						userPhone = phone
					}
				}
				if userMailVal != nil {
					if mail, ok := userMailVal.(string); ok {
						userMail = mail
					}
				}

				// Refresh user info from Warden
				if userPhone != "" || userMail != "" {
					refreshStart := time.Now()
					refreshCtx, refreshSpan := tracing.StartSpan(forwardAuthCtx, "warden.refresh_user_info")
					refreshSpan.SetAttributes(
						attribute.String("warden.identifier_type", func() string {
							if userPhone != "" {
								return "phone"
							}
							return "mail"
						}()),
					)

					userInfo := auth.GetUserInfo(refreshCtx, userPhone, userMail)
					refreshDuration := time.Since(refreshStart)

					if userInfo != nil {
						// Update session with fresh authorization info
						if len(userInfo.Scope) > 0 {
							sess.Set("user_scope", userInfo.Scope)
						}
						if userInfo.Role != "" {
							sess.Set("user_role", userInfo.Role)
						}
						sess.Set("auth_refreshed_at", time.Now().Unix())
						if err := sess.Save(); err != nil {
							tracing.RecordError(refreshSpan, err)
							metrics.RecordAuthRefresh("failure", refreshDuration)
							logrus.Warnf("Failed to save session after auth refresh: %v", err)
						} else {
							refreshSpan.SetAttributes(attribute.Bool("warden.refresh_success", true))
							metrics.RecordAuthRefresh("success", refreshDuration)
							logrus.Debugf("Auth info refreshed for user: phone=%s, mail=%s", userPhone, userMail)
						}
					} else {
						refreshSpan.SetAttributes(attribute.Bool("warden.refresh_success", false))
						metrics.RecordAuthRefresh("failure", refreshDuration)
						logrus.Warnf("Failed to refresh auth info: user not found in Warden")
					}
					refreshSpan.End()
				}
			}
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

		// Set AMR (Authentication Method Reference) header
		userAMRVal := sess.Get("user_amr")
		if userAMRVal != nil {
			if amr, ok := userAMRVal.([]string); ok && len(amr) > 0 {
				ctx.Set("X-Auth-AMR", strings.Join(amr, ","))
			} else if amr, ok := userAMRVal.([]interface{}); ok && len(amr) > 0 {
				// Handle case where AMR is stored as []interface{}
				amrStrs := make([]string, 0, len(amr))
				for _, a := range amr {
					if str, ok := a.(string); ok {
						amrStrs = append(amrStrs, str)
					}
				}
				if len(amrStrs) > 0 {
					ctx.Set("X-Auth-AMR", strings.Join(amrStrs, ","))
				}
			}
		}

		forwardAuthSpan.SetAttributes(attribute.Bool("auth.authenticated", true))
		if userID != "" {
			forwardAuthSpan.SetAttributes(attribute.String("auth.user_id", userID))
		}

		return ctx.SendStatus(fiber.StatusOK)
	}
}
