package handlers

import (
	"context"
	"crypto/subtle"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	forwardauth "github.com/soulteary/forwardauth-kit/v2"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/tracing-kit"
	"go.opentelemetry.io/otel/attribute"
)

func allowTrustedHeaderAuth(ctx fiber.Ctx) bool {
	if !config.HeaderAuthEnabled.ToBool() {
		return false
	}
	expected := config.HeaderAuthSharedSecret.String()
	provided := ctx.Get(config.HeaderAuthSecretHeader.String())
	if expected == "" || len(expected) != len(provided) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func sanitizeTrustedIdentityHeaders(ctx fiber.Ctx) {
	if !allowTrustedHeaderAuth(ctx) {
		ctx.Request().Header.Del("X-User-Phone")
		ctx.Request().Header.Del("X-User-Mail")
	}
	// The proxy credential is only for Stargate and must never be propagated.
	ctx.Request().Header.Del(config.HeaderAuthSecretHeader.String())
}

func handleNotAuthenticated(ctx fiber.Ctx) error {
	if IsHTMLRequest(ctx) {
		return ctx.Redirect().Status(fiber.StatusFound).To(BuildCallbackURL(ctx))
	}
	return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.auth_required"))
}

// SessionStoreForCheck is the minimal interface required by CheckRoute to get session.
// *session.Store implements it; tests can pass a mock to simulate store.Get failure.
type SessionStoreForCheck interface {
	Get(ctx fiber.Ctx) (*session.Session, error)
}

// CheckRoute is the main authentication check handler for Traefik Forward Auth.
// It uses forwardauth-kit to handle authentication logic.
//
// On successful authentication, it sets the X-Forwarded-User header (or configured header name)
// and returns 200 OK. On failure, it either redirects to login (HTML) or returns 401 (API).
//
// Parameters:
//   - store: Session store (or mock implementing SessionStoreForCheck) for managing user sessions
//
// Returns a Fiber handler function.
func CheckRoute(store SessionStoreForCheck) func(c fiber.Ctx) error {
	// Get the ForwardAuth handler
	handler := GetForwardAuthHandler()
	if handler == nil {
		// Fallback: handler not initialized, return error
		return func(ctx fiber.Ctx) error {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, "ForwardAuth handler not initialized")
		}
	}

	return func(ctx fiber.Ctx) error {
		sanitizeTrustedIdentityHeaders(ctx)
		// Get trace context from middleware
		traceCtx := ctx.Locals("trace_context")
		if traceCtx == nil {
			traceCtx = ctx.Context()
		}
		spanCtx := traceCtx.(context.Context)

		// Start span for forward auth check
		_, forwardAuthSpan := tracing.StartSpan(spanCtx, "auth.forward_auth")
		defer forwardAuthSpan.End()

		forwardAuthSpan.SetAttributes(
			attribute.String("http.path", ctx.Path()),
			attribute.String("http.method", ctx.Method()),
		)

		// Store trace context for forwardauth-kit to use
		ctx.Locals("trace_context", spanCtx)

		// Get session
		sess, err := store.Get(ctx)
		if err != nil {
			tracing.RecordError(forwardAuthSpan, err)
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
		}
		defer sess.Release()
		refreshed := false
		if auth.IsAuthenticated(sess) {
			refreshed, err = refreshAuthorizationIfNeeded(spanCtx, sess)
			if err != nil {
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.auth_required"))
			}
		}
		if requiresStepUp(ctx, sess) {
			return redirectToStepUp(ctx)
		}

		// Wrap Fiber context and session for forwardauth-kit
		faCtx := forwardauth.NewFiberContext(ctx)
		faSess := forwardauth.NewFiberSession(sess)

		// Perform authentication check using forwardauth-kit
		result, err := handler.Check(faCtx, faSess)
		if refreshed {
			if saveErr := sess.Save(); saveErr != nil {
				tracing.RecordError(forwardAuthSpan, saveErr)
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
			}
		}
		if err != nil {
			forwardAuthSpan.SetAttributes(attribute.Bool("auth.authenticated", false))

			switch err {
			case forwardauth.ErrInvalidPassword:
				if limitErr := rateLimitPasswordHeaderFailure(ctx); limitErr != nil {
					return limitErr
				}
				return handleNotAuthenticated(ctx)
			case forwardauth.ErrNotAuthenticated, forwardauth.ErrUserNotFound:
				return handleNotAuthenticated(ctx)
			case forwardauth.ErrStepUpRequired:
				return handler.HandleStepUpRequired(faCtx)
			case forwardauth.ErrSessionRequired:
				return handleNotAuthenticated(ctx)
			default:
				return handleNotAuthenticated(ctx)
			}
		}

		// Header and password authentication are stateless. They are evaluated by
		// forwardauth-kit after the session-only check above, so enforce the path
		// policy again after a successful authentication result. A protected path
		// requires a session because that is where the recent verification marker
		// is stored.
		if stepUpApplies(ctx) {
			if !stepUpVerified(sess) {
				if auth.IsAuthenticated(sess) {
					return redirectToStepUp(ctx)
				}
				return SendErrorResponse(ctx, fiber.StatusForbidden, "step-up requires an authenticated session")
			}
			// A verification marker belongs to the session identity that created it.
			// Password and trusted-header checkers run before the session checker, so
			// never let a stateless result borrow a verified session's marker.
			if result.AuthMethod != forwardauth.AuthMethodSession {
				return SendErrorResponse(ctx, fiber.StatusForbidden, "step-up verification does not match the authentication identity")
			}
		}

		// Set authentication headers
		handler.SetAuthHeaders(faCtx, result)

		// Record tracing attributes
		forwardAuthSpan.SetAttributes(attribute.Bool("auth.authenticated", true))
		if result.UserID != "" {
			forwardAuthSpan.SetAttributes(attribute.String("auth.user_id", result.UserID))
		}
		if result.AuthMethod.String() != "none" {
			forwardAuthSpan.SetAttributes(attribute.String("auth.method", result.AuthMethod.String()))
		}

		return ctx.SendStatus(fiber.StatusOK)
	}
}
