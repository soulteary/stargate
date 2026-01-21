package handlers

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"

	"github.com/soulteary/stargate/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
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
		userPhone := ctx.FormValue("phone")
		userMail := ctx.FormValue("mail")

		// Check if at least one identifier is provided
		if userPhone == "" && userMail == "" {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T("error.user_not_in_list"))
		}

		// Check if Herald is enabled
		if !config.HeraldEnabled.ToBool() {
			return SendErrorResponse(ctx, fiber.StatusBadRequest, "验证码服务未配置")
		}

		// Step 1: Get complete user information from Warden (as per Claude.md spec)
		// This ensures we use the official email/phone from Warden, not user input
		userInfo := auth.GetUserInfo(ctx.Context(), userPhone, userMail)
		if userInfo == nil {
			logrus.Warnf("User not found in Warden or not active: phone=%s, mail=%s", userPhone, userMail)
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.user_not_in_list"))
		}

		// Step 2: Use user_id from Warden if available, otherwise generate one
		userID := userInfo.UserID
		if userID == "" {
			userID = generateUserID(userInfo.Phone, userInfo.Mail)
		}

		// Step 3: Determine channel and destination from Warden data
		// Use Warden's official email/phone as destination (not user input)
		channel := "email"
		destination := userInfo.Mail
		if userInfo.Phone != "" {
			channel = "sms"
			destination = userInfo.Phone
		} else if destination == "" {
			// Fallback: if Warden doesn't provide destination, use user input
			// This should not happen if Warden is properly configured
			logrus.Warnf("Warden user info missing destination, using user input: phone=%s, mail=%s", userPhone, userMail)
			destination = userMail
			if userPhone != "" {
				channel = "sms"
				destination = userPhone
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
				return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "验证码服务暂时不可用，请使用 OTP 验证")
			}
			return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "验证码服务暂时不可用，请稍后重试")
		}

		// Step 5: Create challenge via Herald
		createReq := &herald.CreateChallengeRequest{
			UserID:      userID,
			Channel:     channel,
			Destination: destination,
			Purpose:     "login",
			Locale:      locale,
			ClientIP:    ctx.IP(),
			UA:          ctx.Get("User-Agent"),
		}

		createResp, err := heraldClient.CreateChallenge(ctx.Context(), createReq)
		if err != nil {
			logrus.Errorf("Failed to create challenge: %v", err)

			// Check if it's a connection error (Herald service unavailable)
			if heraldErr, ok := err.(*herald.HeraldError); ok {
				if heraldErr.StatusCode == 0 || heraldErr.Reason == "connection_failed" {
					// Herald service is unavailable, suggest OTP fallback if enabled
					otpEnabled := config.WardenOTPEnabled.ToBool()
					if otpEnabled {
						return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "验证码服务暂时不可用，请使用 OTP 验证")
					}
					return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "验证码服务暂时不可用，请稍后重试")
				}
				// Other errors (rate limit, etc.)
				if heraldErr.StatusCode == http.StatusTooManyRequests {
					return SendErrorResponse(ctx, fiber.StatusTooManyRequests, "请求过于频繁，请稍后重试")
				}
			}

			// Default error handling
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, "发送验证码失败: "+err.Error())
		}

		// Return success response with challenge_id
		ctx.Set("Content-Type", "application/json")
		return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":      true,
			"message":      "验证码已发送",
			"challenge_id": createResp.ChallengeID,
			"expires_in":   createResp.ExpiresIn,
		})
	}
}
