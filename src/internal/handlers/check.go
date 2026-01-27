package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	forwardauth "github.com/soulteary/forwardauth-kit"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/tracing-kit"
	"go.opentelemetry.io/otel/attribute"
)

// CheckRoute is the main authentication check handler for Traefik Forward Auth.
// It uses forwardauth-kit to handle authentication logic.
//
// On successful authentication, it sets the X-Forwarded-User header (or configured header name)
// and returns 200 OK. On failure, it either redirects to login (HTML) or returns 401 (API).
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func CheckRoute(store *session.Store) func(c *fiber.Ctx) error {
	// Get the ForwardAuth handler
	handler := GetForwardAuthHandler()
	if handler == nil {
		// Fallback: handler not initialized, return error
		return func(ctx *fiber.Ctx) error {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, "ForwardAuth handler not initialized")
		}
	}

	return func(ctx *fiber.Ctx) error {
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

		// Wrap Fiber context and session for forwardauth-kit
		faCtx := forwardauth.NewFiberContext(ctx)
		faSess := forwardauth.NewFiberSession(sess)

		// Perform authentication check using forwardauth-kit
		result, err := handler.Check(faCtx, faSess)
		if err != nil {
			forwardAuthSpan.SetAttributes(attribute.Bool("auth.authenticated", false))

			switch err {
			case forwardauth.ErrNotAuthenticated, forwardauth.ErrInvalidPassword, forwardauth.ErrUserNotFound:
				return handler.HandleNotAuthenticated(faCtx)
			case forwardauth.ErrStepUpRequired:
				return handler.HandleStepUpRequired(faCtx)
			case forwardauth.ErrSessionRequired:
				return handler.HandleNotAuthenticated(faCtx)
			default:
				return handler.HandleNotAuthenticated(faCtx)
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
