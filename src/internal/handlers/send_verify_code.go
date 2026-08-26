package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"

	"github.com/soulteary/herald/pkg/herald"
	secure "github.com/soulteary/secure-kit"
	"github.com/soulteary/stargate/src/internal/auditlog"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/metrics"
	"github.com/soulteary/tracing-kit"
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

// getLocaleFromConfig converts language code to locale format
// e.g., "en" -> "en-US", "zh" -> "zh-CN"
func getLocaleFromConfig() string {
	lang := strings.ToLower(config.Language.String())
	switch lang {
	case "zh":
		return "zh-CN"
	case "en":
		return "en-US"
	case "fr":
		return "fr-FR"
	case "it":
		return "it-IT"
	case "ja":
		return "ja-JP"
	case "de":
		return "de-DE"
	case "ko":
		return "ko-KR"
	default:
		return "en-US"
	}
}

// SendVerifyCodeAPI handles POST requests to /_send_verify_code for sending verification codes via Herald
func SendVerifyCodeAPI() func(c fiber.Ctx) error {
	return func(ctx fiber.Ctx) error {
		// Get trace context from middleware
		traceCtx := ctx.Locals("trace_context")
		if traceCtx == nil {
			traceCtx = ctx.Context()
		}
		spanCtx := traceCtx.(context.Context)

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
		canonicalPhone := strings.TrimSpace(userInfo.Phone)
		canonicalMail := strings.TrimSpace(userInfo.Mail)
		canonicalDingTalkID := strings.TrimSpace(userInfo.DingtalkUserID)

		var channel, destination string
		deliverVia := ctx.FormValue("deliver_via")
		if deliverVia == "sms" && !config.LoginSMSEnabled.ToBool() {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.login_sms_disabled"), "channel_disabled")
		}
		if deliverVia == "email" && !config.LoginEmailEnabled.ToBool() {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.login_email_disabled"), "channel_disabled")
		}
		switch deliverVia {
		case "dingtalk":
			if canonicalDingTalkID != "" {
				channel, destination = "dingtalk", canonicalDingTalkID
			} else if canonicalPhone != "" {
				// Herald may resolve the DingTalk user by the canonical Warden phone.
				channel, destination = "dingtalk", canonicalPhone
			} else {
				return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.dingtalk_not_bound"), "dingtalk_not_bound")
			}
		case "email":
			if config.LoginEmailEnabled.ToBool() && canonicalMail != "" {
				channel, destination = "email", canonicalMail
			} else if config.LoginSMSEnabled.ToBool() && canonicalPhone != "" {
				channel, destination = "sms", canonicalPhone
			} else {
				return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.user_not_in_list"), "destination_unavailable")
			}
		case "sms":
			if config.LoginSMSEnabled.ToBool() && canonicalPhone != "" {
				channel, destination = "sms", canonicalPhone
			} else if config.LoginEmailEnabled.ToBool() && canonicalMail != "" {
				channel, destination = "email", canonicalMail
			} else {
				return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.user_not_in_list"), "destination_unavailable")
			}
		default:
			if config.LoginSMSEnabled.ToBool() && canonicalPhone != "" {
				channel, destination = "sms", canonicalPhone
			} else if config.LoginEmailEnabled.ToBool() && canonicalMail != "" {
				channel, destination = "email", canonicalMail
			} else {
				return sendVerifyCodeErrorJSON(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.user_not_in_list"), "destination_unavailable")
			}
		}

		// Step 4: Get locale from config or Accept-Language header
		locale := getLocaleFromConfig()
		acceptLang := ctx.Get("Accept-Language")
		if acceptLang != "" {
			// Parse Accept-Language header (simple parsing, takes first language)
			// Format: "en-US,en;q=0.9" -> "en-US"
			parts := strings.Split(acceptLang, ",")
			if len(parts) > 0 {
				langPart := strings.TrimSpace(parts[0])
				// Remove quality value if present
				if idx := strings.Index(langPart, ";"); idx >= 0 {
					langPart = langPart[:idx]
				}
				if langPart != "" {
					locale = langPart
				}
			}
		}

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
		createResp, err := heraldClient.CreateChallenge(heraldCtx, createReq)
		heraldDuration := time.Since(heraldStartTime)
		if err != nil {
			tracing.RecordError(heraldSpan, err)
			heraldSpan.End()
			log.Error().Err(err).Msg("Failed to create challenge")

			reason := "unknown_error"
			// Check if it's a connection error (Herald service unavailable)
			if heraldErr, ok := err.(*herald.HeraldError); ok {
				if heraldErr.StatusCode == 0 || heraldErr.Reason == "connection_failed" {
					reason = "connection_failed"
					// Herald service is unavailable, suggest OTP fallback if enabled
					otpEnabled := config.HeraldTOTPEnabled.ToBool()
					if otpEnabled {
						auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), false, reason)
						return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_use_otp"), reason)
					}
					auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), false, reason)
					return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"), reason)
				}
				// Other errors (rate limit, etc.)
				if heraldErr.StatusCode == http.StatusTooManyRequests {
					reason = "rate_limited"
					auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), false, reason)
					return sendVerifyCodeErrorJSON(ctx, fiber.StatusTooManyRequests, i18n.T(ctx, "error.rate_limited_retry"), reason)
				}
				reason = heraldErr.Reason
			}

			// Default error handling
			auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), false, reason)
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusInternalServerError, i18n.Tf(ctx, "error.send_verify_code_failed", err.Error()), reason)
		}

		// Log successful verification code send
		metrics.RecordHeraldCall("create_challenge", "success", heraldDuration)
		auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), true, "")

		heraldSpan.SetAttributes(
			attribute.String("herald.challenge_id", createResp.ChallengeID),
			attribute.Int("herald.expires_in", createResp.ExpiresIn),
			attribute.String("herald.result", "success"),
		)
		heraldSpan.End()

		sendCodeSpan.SetAttributes(
			attribute.String("auth.user_id", userID),
			attribute.String("auth.channel", channel),
			attribute.String("auth.result", "success"),
		)

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
		// When DEBUG=true and Herald returns debug_code (HERALD_TEST_MODE), pass through for display/autofill
		if config.Debug.ToBool() && createResp.DebugCode != "" {
			resp["debug_code"] = createResp.DebugCode
		}
		ctx.Set("Content-Type", "application/json")
		return ctx.Status(fiber.StatusOK).JSON(resp)
	}
}
