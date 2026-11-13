package handlers

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	forwardauth "github.com/soulteary/forwardauth-kit/v3"
	fafiber "github.com/soulteary/forwardauth-kit/v3/fiberadapter"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/tracing-kit/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

func sanitizeTrustedIdentityHeaders(ctx fiber.Ctx) bool {
	trusted := allowTrustedHeaderAuth(ctx)
	if !trusted {
		ctx.Request().Header.Del("X-User-Phone")
		ctx.Request().Header.Del("X-User-Mail")
	}
	// The proxy credential is only for Stargate and must never be propagated.
	ctx.Request().Header.Del(config.HeaderAuthSecretHeader.String())
	return trusted
}

var lookupTrustedHeaderUser = auth.LookupUserInfo

func trustedHeaderAuthResult(ctx context.Context, phone, mail string) (*forwardauth.AuthResult, time.Time, error) {
	if phone == "" && mail == "" {
		return nil, time.Time{}, nil
	}
	userInfo, deadline, lookupErr := lookupTrustedHeaderUser(ctx, phone, mail)
	if lookupErr != nil && !errors.Is(lookupErr, auth.ErrUserUnavailable) {
		return nil, time.Time{}, errAuthorizationUnavailable
	}
	if userInfo == nil {
		return nil, time.Time{}, forwardauth.ErrUserNotFound
	}
	return &forwardauth.AuthResult{
		Authenticated: true,
		AuthMethod:    forwardauth.AuthMethodHeader,
		UserID:        userInfo.UserID,
		Email:         userInfo.Mail,
		Phone:         userInfo.Phone,
		Name:          userInfo.Name,
		Scopes:        userInfo.Scope,
		Role:          userInfo.Role,
	}, deadline, nil
}

func handleNotAuthenticated(ctx fiber.Ctx) error {
	// This is where ignored forwarded headers show: the redirect below is
	// built from the request Stargate received, not the one the user made.
	warnUntrustedForwardedHeaders(ctx)
	if IsHTMLRequest(ctx) {
		return ctx.Redirect().Status(fiber.StatusFound).To(BuildCallbackURL(ctx))
	}
	return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.auth_required"))
}

// SessionStoreForCheck is the minimal interface required by CheckRoute to get a
// session and to confirm that a refreshed session still exists after it is saved.
// *session.Store implements it; tests can pass a mock to simulate store.Get failure.
type SessionStoreForCheck interface {
	Get(ctx fiber.Ctx) (*session.Session, error)
	sessionByID
}

// CheckRoute is the main authentication check handler for Traefik Forward Auth.
// It uses forwardauth-kit to handle authentication logic.
//
// On successful authentication, it sets the X-Forwarded-User header (or configured header name)
// and returns 200 OK. Authentication failures redirect to login (HTML) or return
// 401 (API); unavailable directory authorization returns 503 without revoking the session.
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
		// Never reflect a client-provided verification declaration, including on
		// authentication failures. Only our successfully checked session can set it.
		ctx.Request().Header.Del(verifiedEmailHeader)
		ctx.Response().Header.Del(verifiedEmailHeader)
		trustedIdentity := sanitizeTrustedIdentityHeaders(ctx)
		// Tracing publishes its span through Fiber's standard Context/SetContext
		// API; when tracing is disabled this still carries the request deadline.
		spanCtx := ctx.Context()

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

		// Wrap Fiber context and session for forwardauth-kit
		faCtx := fafiber.NewContext(ctx)
		faSess := fafiber.NewSession(sess)

		// Password authentication keeps its original priority over trusted
		// identity headers. Otherwise resolve the trusted identity here with the
		// active request context, before falling back to the session checker.
		var result *forwardauth.AuthResult
		var headerDeadline time.Time
		passwordAttempt := config.PasswordHeaderAuthEnabled.ToBool() && ctx.Get("Stargate-Password") != ""
		if trustedIdentity && !passwordAttempt {
			result, headerDeadline, err = trustedHeaderAuthResult(spanCtx, ctx.Get("X-User-Phone"), ctx.Get("X-User-Mail"))
			if result == nil && err == nil {
				result, err = handler.Check(faCtx, faSess)
			}
		} else {
			result, err = handler.Check(faCtx, faSess)
		}
		// A cookie cannot override an explicit authentication method. Refresh
		// directory authorization only after the session checker is selected.
		if err == nil && result.AuthMethod == forwardauth.AuthMethodSession {
			refreshed, err = refreshAuthorizationIfNeeded(spanCtx, sess)
			if err != nil {
				if errors.Is(err, errAuthorizationUnavailable) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "directory authorization is unavailable")
				}
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.auth_required"))
			}
			if requiresStepUp(ctx, sess) {
				return redirectToStepUp(ctx)
			}
			if refreshed {
				// Rebuild the result from refreshed claims before publishing headers.
				result, err = handler.Check(faCtx, faSess)
			}
		}
		if refreshed {
			if saveErr := saveLiveSession(spanCtx, store, sess); saveErr != nil {
				tracing.RecordError(forwardAuthSpan, saveErr)
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
			}
		}
		if err != nil {
			forwardAuthSpan.SetAttributes(attribute.Bool("auth.authenticated", false))
			return handleForwardAuthFailure(ctx, handler, faCtx, err)
		}

		// Header and password authentication are stateless. They are evaluated by
		// the selected checker above, so enforce the path
		// policy again after a successful authentication result. A protected path
		// requires a session because that is where the recent verification marker
		// is stored.
		if allowed, stepUpErr := enforceStepUpResult(ctx, sess, result); !allowed {
			return stepUpErr
		}

		// Recheck after authentication: processing must not carry a session past
		// the source deadline observed at the start of this request.
		if result.AuthMethod == forwardauth.AuthMethodHeader && config.WardenSourceMaxAge.ToDuration() > 0 && (headerDeadline.IsZero() || !authorizationNow().Before(headerDeadline)) {
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "directory authorization is unavailable")
		}
		if result.AuthMethod == forwardauth.AuthMethodSession && wardenSessionSourceExpired(sess) {
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "directory authorization is unavailable")
		}

		// Set authentication headers
		handler.SetAuthHeaders(faCtx, result)
		setVerifiedEmailHeader(ctx, sess, result, time.Now())

		recordForwardAuthResult(forwardAuthSpan, result)

		return ctx.SendStatus(fiber.StatusOK)
	}
}

func handleForwardAuthFailure(ctx fiber.Ctx, handler *forwardauth.Handler, faCtx *fafiber.Context, err error) error {
	switch err {
	case errAuthorizationUnavailable:
		return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "directory authorization is unavailable")
	case forwardauth.ErrInvalidPassword:
		if limited, limitErr := rateLimitPasswordHeaderFailure(ctx); limited || limitErr != nil {
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

func enforceStepUpResult(ctx fiber.Ctx, sess *session.Session, result *forwardauth.AuthResult) (bool, error) {
	if stepUpApplies(ctx) {
		if !stepUpVerified(sess) {
			if auth.IsAuthenticated(sess) {
				return false, redirectToStepUp(ctx)
			}
			return false, SendErrorResponse(ctx, fiber.StatusForbidden, "step-up requires an authenticated session")
		}
		// A verification marker belongs to the session identity that created it.
		// Password and trusted-header checkers run before the session checker, so
		// never let a stateless result borrow a verified session's marker.
		if result.AuthMethod != forwardauth.AuthMethodSession {
			return false, SendErrorResponse(ctx, fiber.StatusForbidden, "step-up verification does not match the authentication identity")
		}
	}

	return true, nil
}

func recordForwardAuthResult(forwardAuthSpan trace.Span, result *forwardauth.AuthResult) {
	// Record tracing attributes
	forwardAuthSpan.SetAttributes(attribute.Bool("auth.authenticated", true))
	if result.UserID != "" {
		forwardAuthSpan.SetAttributes(attribute.String("auth.user_id", result.UserID))
	}
	if result.AuthMethod.String() != "none" {
		forwardAuthSpan.SetAttributes(attribute.String("auth.method", result.AuthMethod.String()))
	}
}
