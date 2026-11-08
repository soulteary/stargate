package handlers

import (
	"github.com/gofiber/fiber/v3"
	i18nfiber "github.com/soulteary/i18n-kit/v4/fiberadapter"
)

func authenticationReasonAllowed(reason string) bool {
	switch reason {
	case "identifier_required", "identifier_invalid", "verification_required", "invalid", "expired", "locked", "too_many_attempts", "rate_limited", "lock_contention", "service_unavailable", "authentication_failed", "totp_not_enrolled":
		return true
	default:
		return false
	}
}

func authenticationUILanguage(c fiber.Ctx) string { return string(i18nfiber.Language(c)) }
