package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	i18nkit "github.com/soulteary/i18n-kit/v4"
	i18nfiber "github.com/soulteary/i18n-kit/v4/fiberadapter"
	"github.com/soulteary/stargate/src/internal/config"
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
	lang := i18nfiber.Language(c)
	if c.Locals(i18nkit.LocalsLanguageKey) == nil {
		lang = i18nfiber.DetectWith(i18nkit.NewDetector(i18nkit.DetectorConfig{Default: i18n.GetLanguage()}), c)
	}
	switch lang {
	case i18n.LangEN, i18n.LangZH, i18n.LangFR, i18n.LangDE, i18n.LangIT, i18n.LangJA, i18n.LangKO:
		return string(lang)
	default:
		return "en"
	}
}

// authenticationUIData is serialized into an HTML attribute, never executable JS.
func authenticationUIData(c fiber.Ctx) (map[string]string, string) {
	words := i18n.AuthenticationUI(i18n.Language(authenticationUILanguage(c)))
	encoded, _ := json.Marshal(words) // string-only map has no failing values
	return words, string(encoded)
}

// uniformLoginPage reports whether the login page follows uniform responses:
// the mode is on and the deployment offers Warden sign-in through Herald.
func uniformLoginPage() bool {
	return uniformResponses() && config.WardenEnabled.ToBool() && config.HeraldEnabled.ToBool()
}

// loginUIData is authenticationUIData for the login page. With uniform
// responses the server does not say whether a code was sent or why a sign-in
// failed, and it handles a request by one identifier, so the phrases that
// would say otherwise are replaced. The other pages keep the original
// vocabulary.
func loginUIData(c fiber.Ctx) (map[string]string, string) {
	words, encoded := authenticationUIData(c)
	if !uniformLoginPage() {
		return words, encoded
	}
	for key, phrase := range i18n.UniformAuthenticationUI(i18n.Language(authenticationUILanguage(c))) {
		words[key] = phrase
	}
	replaced, _ := json.Marshal(words) // string-only map has no failing values
	return words, string(replaced)
}
