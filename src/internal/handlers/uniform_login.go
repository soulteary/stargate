package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/tracing-kit/v2"
)

// authenticateWardenUniform is Warden sign-in in uniform mode.
//
// Requests that are malformed, or that the configuration cannot serve, are
// answered before the directory lookup and therefore for every identifier.
// The existing logic then runs unchanged, and every failure it reports is
// replaced by one response. A failed directory lookup is the exception: it
// does not depend on the identifier being listed and keeps its own answer.
func (identity *loginIdentity) authenticateWardenUniform(ctx fiber.Ctx, loginCtx context.Context, loginSpan trace.Span) (bool, error) {
	// A phone number, when supplied, is the only identifier. With the default
	// fallback the owner of a listed mail could sign in with it only next to
	// an unlisted phone, and so learn which phones are listed.
	if identity.userPhone != "" {
		identity.userMail = ""
	}
	if answered, err := identity.uniformLoginPrecheck(ctx, loginSpan); answered {
		return false, err
	}
	return replaceLoginFailure(ctx, func() (bool, bool, error) {
		authenticated, err := identity.authenticateWarden(ctx, loginCtx, loginSpan)
		return authenticated, identity.lookupFailed, err
	})
}

// replaceLoginFailure runs attempt, which reports whether it authenticated
// and whether its own answer must be kept. When neither holds, whatever the
// attempt wrote is discarded and the single failure of uniform sign-in is
// sent instead.
//
// The response headers are restored to what the middleware had set before the
// attempt, so that no status, Content-Type, Retry-After or cookie written by a
// failure path survives. Replacing the response here, rather than at each
// place that reports a failure, keeps a failure added later from leaking.
func replaceLoginFailure(ctx fiber.Ctx, attempt func() (authenticated, keep bool, err error)) (bool, error) {
	var before fasthttp.ResponseHeader
	ctx.Response().Header.CopyTo(&before)

	authenticated, keep, err := attempt()
	if authenticated || keep {
		return authenticated, err
	}

	ctx.Response().ResetBody()
	before.CopyTo(&ctx.Response().Header)
	return false, SendAuthenticationError(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.login_failed_uniform"), "authentication_failed")
}

// uniformLoginPrecheck answers the requests whose outcome does not depend on
// the identifier, with the responses the default mode gives a listed user. It
// reports whether it wrote a response.
func (identity *loginIdentity) uniformLoginPrecheck(ctx fiber.Ctx, loginSpan trace.Span) (bool, error) {
	if identity.userPhone != "" && !auth.IsValidPhone(identity.userPhone) {
		tracing.RecordError(loginSpan, fmt.Errorf("invalid phone format"))
		return true, SendAuthenticationError(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.invalid_phone_format"), "identifier_invalid")
	}
	if identity.userPhone == "" && identity.userMail == "" {
		tracing.RecordError(loginSpan, fmt.Errorf("no identifier provided"))
		return true, SendAuthenticationError(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.user_not_in_list"), "identifier_required")
	}

	otpEnabled := config.HeraldTOTPEnabled.ToBool()
	if ctx.FormValue("use_otp") == "true" {
		if !otpEnabled {
			return true, SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.provide_verify_code_or_otp"))
		}
		if ctx.FormValue("otp_code") == "" {
			return true, SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.otp_code_required"))
		}
		if getHeraldClient() == nil {
			return true, SendAuthenticationError(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"), "service_unavailable")
		}
	} else {
		if !config.HeraldEnabled.ToBool() {
			if !otpEnabled {
				return true, SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.herald_not_configured_use_otp_or_contact"))
			}
			return true, SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.herald_not_configured_use_otp"))
		}
		challengeID := ctx.FormValue("challenge_id")
		if challengeID == "" || ctx.FormValue("verify_code") == "" {
			return true, SendAuthenticationError(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.verify_code_and_challenge_required"), "verification_required")
		}
		if getHeraldClient() == nil {
			return true, SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.herald_unavailable"))
		}
		// The context is read for every identifier, so a store failure is
		// reported whether or not the identifier is listed.
		if _, err := loadChallengeContext(ctx.Context(), challengeID); err != nil {
			log.Error().Err(err).Str("challenge_id", challengeID).Msg("Failed to load challenge audit context")
			return true, SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "verification context is unavailable")
		}
	}
	return false, nil
}
