package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"

	"github.com/soulteary/herald/pkg/herald"
	secure "github.com/soulteary/secure-kit/v2"
	"github.com/soulteary/stargate/src/internal/auditlog"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/metrics"
	"github.com/soulteary/tracing-kit/v2"
	"github.com/soulteary/warden/pkg/warden"
)

// sendVerifyCodeErrorJSON returns JSON in the shape expected by the login page Send Code UI:
// { success: false, message: "...", reason: "..." }. Use this for all error paths of /_send_verify_code
// so the front-end can display result.message and result.reason correctly.
func sendVerifyCodeErrorJSON(ctx fiber.Ctx, statusCode int, message, reason string) error {
	ctx.Set("Content-Type", "application/json")
	return ctx.Status(statusCode).JSON(fiber.Map{
		"success": false,
		"message": message,
		"reason":  reason,
	})
}

// selectVerificationDestination resolves the same canonical Warden-backed
// channel and destination for challenge creation and verification auditing.
// Request identifiers are lookup inputs only and are never used as fallback
// destinations.
func selectVerificationDestination(userInfo *warden.AllowListUser, deliverVia string) (channel, destination, reason string) {
	if userInfo == nil {
		return "", "", "destination_unavailable"
	}
	canonicalPhone := strings.TrimSpace(userInfo.Phone)
	canonicalMail := strings.TrimSpace(userInfo.Mail)
	canonicalDingTalkID := strings.TrimSpace(userInfo.DingtalkUserID)

	if deliverVia == "sms" && !config.LoginSMSEnabled.ToBool() {
		return "", "", "channel_disabled"
	}
	if deliverVia == "email" && !config.LoginEmailEnabled.ToBool() {
		return "", "", "channel_disabled"
	}
	if deliverVia == "dingtalk" {
		if config.LoginDingtalkRequireUserID.ToBool() {
			// The directory binding is one opaque userid, never a receiver list
			// or a phone-number lookup. Reject CR/LF before trimming as well.
			if canonicalDingTalkID == "" || strings.ContainsAny(userInfo.DingtalkUserID, ",\r\n") ||
				strings.IndexFunc(canonicalDingTalkID, unicode.IsControl) >= 0 {
				return "", "", "dingtalk_not_bound"
			}
			return "dingtalk", canonicalDingTalkID, ""
		}
		if canonicalDingTalkID != "" {
			return "dingtalk", canonicalDingTalkID, ""
		}
		if canonicalPhone != "" {
			return "dingtalk", canonicalPhone, ""
		}
		return "", "", "dingtalk_not_bound"
	}
	type candidate struct {
		channel, destination string
		enabled              bool
	}
	candidates := []candidate{{"sms", canonicalPhone, config.LoginSMSEnabled.ToBool()}, {"email", canonicalMail, config.LoginEmailEnabled.ToBool()}}
	if deliverVia == "email" {
		candidates[0], candidates[1] = candidates[1], candidates[0]
	}
	for _, option := range candidates {
		if option.enabled && option.destination != "" {
			return option.channel, option.destination, ""
		}
	}

	return "", "", "destination_unavailable"
}

// SendVerifyCodeAPI handles POST requests to /_send_verify_code for sending verification codes via Herald
func SendVerifyCodeAPI() func(c fiber.Ctx) error {
	return func(ctx fiber.Ctx) error {
		// Tracing publishes its span through Fiber's standard Context/SetContext
		// API; when tracing is disabled this still carries the request deadline.
		spanCtx := ctx.Context()

		// Start span for send verify code
		sendCodeCtx, sendCodeSpan := tracing.StartSpan(spanCtx, "auth.send_verify_code")
		defer sendCodeSpan.End()

		userPhone := auth.NormalizePhone(ctx.FormValue("phone"))
		userMail := ctx.FormValue("mail")

		// 手机号规范化后校验格式（如系统自动填充 "138 0013 8000" 已去空格，此处校验是否为有效号码）
		if userPhone != "" && !auth.IsValidPhone(userPhone) {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.invalid_phone_format"), "invalid_phone_format")
		}
		// Check if at least one identifier is provided
		if userPhone == "" && userMail == "" {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.user_not_in_list"), "identifier_required")
		}

		// Check if Herald is enabled
		if !config.HeraldEnabled.ToBool() {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.herald_not_configured"), "herald_not_configured")
		}

		// Step 1: Get complete user information from Warden
		// This ensures we use the official email/phone from Warden, not user input
		wardenCtx, wardenSpan := tracing.StartSpan(sendCodeCtx, "warden.get_user_info")
		wardenSpan.SetAttributes(
			attribute.String("warden.identifier_type", func() string {
				if userPhone != "" {
					return "phone"
				}
				return "mail"
			}()),
		)
		userInfo := auth.GetUserInfo(wardenCtx, userPhone, userMail)
		if userInfo == nil {
			wardenSpan.SetAttributes(attribute.Bool("warden.user_found", false))
			wardenSpan.End()
			tracing.RecordError(sendCodeSpan, fmt.Errorf("user not found in Warden"))
			log.Warn().Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("User not found in Warden or not active")
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.user_not_in_list"), "user_not_in_list")
		}

		// Step 2: Use user_id from Warden if available, otherwise generate one
		userID := userInfo.UserID
		if userID == "" {
			userID = generateUserID(userInfo.Phone, userInfo.Mail)
		}

		wardenSpan.SetAttributes(
			attribute.Bool("warden.user_found", true),
			attribute.String("warden.user_id", userID),
		)
		wardenSpan.End()

		// Step 3: Select a delivery channel using only contact data returned by
		// Warden. Request identifiers are lookup inputs, not trusted destinations:
		// binding a victim user_id to an attacker-supplied fallback address would
		// turn the verification challenge into an account-takeover primitive.
		deliverVia := ctx.FormValue("deliver_via")
		channel, destination, selectionReason := selectVerificationDestination(userInfo, deliverVia)
		if selectionReason != "" {
			return verificationSelectionError(ctx, deliverVia, selectionReason)
		}

		locale := verificationLocale(ctx)

		// Get Herald client
		heraldClient := getHeraldClient()
		if heraldClient == nil {
			// Herald client not initialized, check if OTP is available as fallback
			otpEnabled := config.HeraldTOTPEnabled.ToBool()
			if otpEnabled {
				return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_use_otp"), "connection_failed")
			}
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"), "connection_failed")
		}

		// Step 5: Create challenge via Herald
		heraldCtx, heraldSpan := tracing.StartSpan(sendCodeCtx, "herald.create_challenge")
		defer heraldSpan.End()
		heraldSpan.SetAttributes(
			attribute.String("herald.user_id", userID),
			attribute.String("herald.channel", channel),
			attribute.String("herald.purpose", "login"),
		)
		// Pass through Idempotency-Key so Herald can deduplicate (CLAUDE.md §13.4)
		if idemKey := ctx.Get("Idempotency-Key"); idemKey != "" {
			heraldCtx = context.WithValue(heraldCtx, herald.IdempotencyKeyContextKey, idemKey)
		}

		createReq := &herald.CreateChallengeRequest{
			UserID:      userID,
			Channel:     channel,
			Destination: destination,
			Purpose:     "login",
			Locale:      locale,
			ClientIP:    ctx.IP(),
			UA:          ctx.Get("User-Agent"),
		}

		heraldStartTime := time.Now()
		createResp, metadata, err := heraldClient.CreateChallengeWithMetadata(heraldCtx, createReq)
		heraldDuration := time.Since(heraldStartTime)
		if err != nil {
			tracing.RecordError(heraldSpan, err)
			log.Error().Err(err).Msg("Failed to create challenge")

			return handleChallengeCreationErrorWithMetadata(ctx, err, metadata, userID, channel, destination)
		}

		// Log successful verification code send
		metrics.RecordHeraldCall("create_challenge", "success", heraldDuration)
		auditlog.LogVerifyCodeSend(ctx.Context(), userID, createResp.ChallengeID, channel, destination, ctx.IP(), true, "")
		binding := challengeContext{
			ChallengeID: createResp.ChallengeID,
			UserID:      userID,
			Channel:     channel,
			Destination: destination,
		}
		if err := bindChallengeContext(ctx.Context(), binding, createResp.ExpiresIn); err != nil {
			log.Error().Err(err).Str("challenge_id", createResp.ChallengeID).Msg("Failed to persist challenge audit context")
			tracing.RecordError(heraldSpan, fmt.Errorf("verification context unavailable"))
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"), "binding_unavailable")
		}

		heraldSpan.SetAttributes(
			attribute.String("herald.challenge_id", createResp.ChallengeID),
			attribute.Int("herald.expires_in", createResp.ExpiresIn),
			attribute.String("herald.result", "success"),
		)

		sendCodeSpan.SetAttributes(
			attribute.String("auth.user_id", userID),
			attribute.String("auth.channel", channel),
			attribute.String("auth.result", "success"),
		)

		return sendChallengeCreated(ctx, createResp)
	}
}

// bindChallengeContext records which user, channel, destination and purpose a
// challenge was issued for. A challenge id that is already bound must carry the
// same binding.
func bindChallengeContext(ctx context.Context, binding challengeContext, expiresIn int) error {
	stored, err := getChallengeContextStore().PutIfAbsent(ctx, binding, challengeContextTTL(expiresIn))
	if err == nil && !stored {
		var existing challengeContext
		var found bool
		existing, found, err = getChallengeContextStore().Get(ctx, binding.ChallengeID)
		if err == nil && (!found || existing.ChallengeID != binding.ChallengeID || existing.UserID != binding.UserID || existing.Channel != binding.Channel ||
			existing.Purpose != binding.Purpose || !sameVerificationDestination(binding.Channel, existing.Destination, binding.Destination)) {
			err = fmt.Errorf("verification context conflict")
		}
	}
	return err
}

func verificationSelectionError(ctx fiber.Ctx, deliverVia, selectionReason string) error {
	switch selectionReason {
	case "channel_disabled":
		if deliverVia == "sms" {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.login_sms_disabled"), selectionReason)
		}
		return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.login_email_disabled"), selectionReason)
	case "dingtalk_not_bound":
		return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.dingtalk_not_bound"), selectionReason)
	default:
		return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.user_not_in_list"), selectionReason)
	}
}

func verificationLocale(ctx fiber.Ctx) string {
	switch authenticationUILanguage(ctx) {
	case "zh":
		return "zh-CN"
	case "fr":
		return "fr-FR"
	case "de":
		return "de-DE"
	case "it":
		return "it-IT"
	case "ja":
		return "ja-JP"
	case "ko":
		return "ko-KR"
	default:
		return "en-US"
	}
}

func handleChallengeCreationError(ctx fiber.Ctx, err error, userID, channel, destination string) error {
	return handleChallengeCreationErrorWithMetadata(ctx, err, herald.CreateChallengeMetadata{}, userID, channel, destination)
}

func handleChallengeCreationErrorWithMetadata(ctx fiber.Ctx, err error, metadata herald.CreateChallengeMetadata, userID, channel, destination string) error {
	reason := "unknown_error"
	// Check if it's a connection error (Herald service unavailable)
	var heraldErr *herald.HeraldError
	if errors.As(err, &heraldErr) {
		if heraldErr.StatusCode == 0 || heraldErr.Reason == "connection_failed" {
			reason = "connection_failed"
			// Herald service is unavailable, suggest OTP fallback if enabled
			otpEnabled := config.HeraldTOTPEnabled.ToBool()
			if otpEnabled {
				auditlog.LogVerifyCodeSend(ctx.Context(), userID, "", channel, destination, ctx.IP(), false, reason)
				return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_use_otp"), reason)
			}
			auditlog.LogVerifyCodeSend(ctx.Context(), userID, "", channel, destination, ctx.IP(), false, reason)
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"), reason)
		}
		// Other errors (rate limit, etc.)
		if heraldErr.StatusCode == http.StatusTooManyRequests {
			reason = "rate_limited"
			auditlog.LogVerifyCodeSend(ctx.Context(), userID, "", channel, destination, ctx.IP(), false, reason)
			if seconds := metadata.RetryAfterSeconds; seconds != nil && *seconds >= 1 && *seconds <= 86400 {
				ctx.Set("Retry-After", strconv.Itoa(*seconds))
				return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"success": false, "message": i18n.T(ctx, "error.rate_limited_retry"), "reason": reason, "next_resend_in": *seconds,
				})
			}
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusTooManyRequests, i18n.T(ctx, "error.rate_limited_retry"), reason)
		}
		reason = "provider_error"
	}

	// Default error handling
	auditlog.LogVerifyCodeSend(ctx.Context(), userID, "", channel, destination, ctx.IP(), false, reason)
	return sendVerifyCodeErrorJSON(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.send_verify_code_failed"), reason)
}

func sendChallengeCreated(ctx fiber.Ctx, createResp *herald.CreateChallengeResponse) error {
	// Return success response with challenge_id and next_resend_in (for frontend resend cooldown)
	resp := fiber.Map{
		"success":      true,
		"message":      i18n.T(ctx, "success.verify_code_sent"),
		"challenge_id": createResp.ChallengeID,
		"expires_in":   createResp.ExpiresIn,
	}
	if createResp.NextResendIn > 0 {
		resp["next_resend_in"] = createResp.NextResendIn
	}
	// When DEBUG=true and Herald returns debug_code (HERALD_TEST_MODE), pass
	// through for display/autofill. The code is equivalent to a credential, so
	// it is never relayed from an HTTPS deployment (COOKIE_SECURE=true), even
	// if DEBUG was left enabled there.
	if debugCodeRelayEnabled() && createResp.DebugCode != "" {
		resp["debug_code"] = createResp.DebugCode
	}
	ctx.Set("Content-Type", "application/json")
	return ctx.Status(fiber.StatusOK).JSON(resp)
}

// debugCodeRelayEnabled reports whether Herald's debug code may be shown to
// the client: only with DEBUG=true over local HTTP (COOKIE_SECURE=false).
func debugCodeRelayEnabled() bool {
	return config.Debug.ToBool() && !config.CookieSecure.ToBool()
}
