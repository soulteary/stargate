package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
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
func SendVerifyCodeAPI() func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		// Get trace context from middleware
		traceCtx := ctx.Locals("trace_context")
		if traceCtx == nil {
			traceCtx = ctx.Context()
		}
		spanCtx := traceCtx.(context.Context)

		// Start span for send verify code
		sendCodeCtx, sendCodeSpan := tracing.StartSpan(spanCtx, "auth.send_verify_code")
		defer sendCodeSpan.End()

		userPhone := ctx.FormValue("phone")
		userMail := ctx.FormValue("mail")

		// Check if at least one identifier is provided
		if userPhone == "" && userMail == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.user_not_in_list"))
		}

		// Check if Herald is enabled
		if !config.HeraldEnabled.ToBool() {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.herald_not_configured"))
		}

		// Step 1: Get complete user information from Warden (as per Claude.md spec)
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
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.user_not_in_list"))
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

		// Step 3: Determine channel and destination from Warden data
		// If user requested DingTalk and has dingtalk_userid, use dingtalk channel; else use Warden's email/phone
		var channel, destination string
		deliverVia := ctx.FormValue("deliver_via")
		if deliverVia == "dingtalk" {
			if strings.TrimSpace(userInfo.DingtalkUserID) != "" {
				channel = "dingtalk"
				destination = strings.TrimSpace(userInfo.DingtalkUserID)
			} else {
				log.Warn().Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("User requested DingTalk but account has no dingtalk_userid")
				ctx.Set("Content-Type", "application/json")
				return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"success": false,
					"message": i18n.T(ctx, "error.dingtalk_not_bound"),
					"reason":  "dingtalk_not_bound",
				})
			}
		} else {
			// Respect deliver_via for sms vs email; default: phone if present else mail
			switch deliverVia {
			case "email":
				channel = "email"
				destination = userInfo.Mail
				if destination == "" {
					destination = userMail
				}
			case "sms":
				channel = "sms"
				destination = userInfo.Phone
				if destination == "" {
					destination = userPhone
				}
			default:
				channel = "email"
				destination = userInfo.Mail
				if userInfo.Phone != "" {
					channel = "sms"
					destination = userInfo.Phone
				} else if destination == "" {
					destination = userMail
					if userPhone != "" {
						channel = "sms"
						destination = userPhone
					}
				}
			}
			if destination == "" {
				log.Warn().Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("Warden user info missing destination, using user input")
				if userPhone != "" {
					channel = "sms"
					destination = userPhone
				} else {
					channel = "email"
					destination = userMail
				}
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
			otpEnabled := config.WardenOTPEnabled.ToBool()
			if otpEnabled {
				return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_use_otp"))
			}
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"))
		}

		// Step 5: Create challenge via Herald
		heraldCtx, heraldSpan := tracing.StartSpan(sendCodeCtx, "herald.create_challenge")
		heraldSpan.SetAttributes(
			attribute.String("herald.user_id", userID),
			attribute.String("herald.channel", channel),
			attribute.String("herald.purpose", "login"),
		)

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
					otpEnabled := config.WardenOTPEnabled.ToBool()
					if otpEnabled {
						auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), false, reason)
						return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_use_otp"))
					}
					auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), false, reason)
					return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"))
				}
				// Other errors (rate limit, etc.)
				if heraldErr.StatusCode == http.StatusTooManyRequests {
					reason = "rate_limited"
					auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), false, reason)
					return SendErrorResponse(ctx, fiber.StatusTooManyRequests, i18n.T(ctx, "error.rate_limited_retry"))
				}
				reason = heraldErr.Reason
			}

			// Default error handling
			auditlog.LogVerifyCodeSend(ctx.Context(), userID, channel, destination, ctx.IP(), false, reason)
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.Tf(ctx, "error.send_verify_code_failed", err.Error()))
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

		// Return success response with challenge_id
		ctx.Set("Content-Type", "application/json")
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":      true,
			"message":      i18n.T(ctx, "success.verify_code_sent"),
			"challenge_id": createResp.ChallengeID,
			"expires_in":   createResp.ExpiresIn,
		})
	}
}
