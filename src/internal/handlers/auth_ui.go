package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	i18nkit "github.com/soulteary/i18n-kit/v4"
	i18nfiber "github.com/soulteary/i18n-kit/v4/fiberadapter"
	"github.com/soulteary/stargate/src/internal/i18n"
)

func authenticationReasonAllowed(reason string) bool {
	switch reason {
	case "identifier_required", "identifier_invalid", "verification_required", "invalid", "expired", "locked", "too_many_attempts", "rate_limited", "lock_contention", "service_unavailable", "authentication_failed", "totp_not_enrolled":
		return true
	default:
		return false
	}
}

func authenticationUILanguage(c fiber.Ctx) string {
	if c.Locals(i18nkit.LocalsLanguageKey) == nil {
		return string(i18nfiber.DetectWith(i18nkit.NewDetector(i18nkit.DetectorConfig{Default: i18n.GetLanguage()}), c))
	}
	return string(i18nfiber.Language(c))
}

// authenticationUIData is serialized into an HTML attribute, never executable JS.
func authenticationUIData(c fiber.Ctx) (map[string]string, string) {
	words := i18n.AuthenticationUI(i18n.Language(authenticationUILanguage(c)))
	encoded, _ := json.Marshal(words) // string-only map has no failing values
	return words, string(encoded)
}
