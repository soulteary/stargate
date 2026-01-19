package config

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/soulteary/stargate/src/internal/i18n"
)

// SessionExpiration is the session expiration time
const SessionExpiration = 24 * time.Hour

var (
	Debug = EnvVariable{
		Name:           "DEBUG",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	AuthHost = EnvVariable{
		Name:           "AUTH_HOST",
		Required:       true,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateNotEmptyString,
	}

	LoginPageTitle = EnvVariable{
		Name:           "LOGIN_PAGE_TITLE",
		Required:       false,
		DefaultValue:   "Stargate - Login",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	LoginPageFooterText = EnvVariable{
		Name:           "LOGIN_PAGE_FOOTER_TEXT",
		Required:       false,
		DefaultValue:   "Copyright © 2024 - Stargate",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	Passwords = EnvVariable{
		Name:           "PASSWORDS",
		Required:       true,
		DefaultValue:   "",
		PossibleValues: []string{"algorithm:pass1|pass2|pass3"},
		Validator:      ValidatePasswords,
	}

	UserHeaderName = EnvVariable{
		Name:           "USER_HEADER_NAME",
		Required:       false,
		DefaultValue:   "X-Forwarded-User",
		PossibleValues: []string{"*"},
		Validator:      ValidateNotEmptyString,
	}

	CookieDomain = EnvVariable{
		Name:           "COOKIE_DOMAIN",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny, // Empty value is also valid (means not setting domain)
	}

	Language = EnvVariable{
		Name:           "LANGUAGE",
		Required:       false,
		DefaultValue:   "en",
		PossibleValues: []string{"en", "zh", "fr", "it", "ja", "de", "ko"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	WardenURL = EnvVariable{
		Name:           "WARDEN_URL",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	WardenAPIKey = EnvVariable{
		Name:           "WARDEN_API_KEY",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	WardenEnabled = EnvVariable{
		Name:           "WARDEN_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	WardenCacheTTL = EnvVariable{
		Name:           "WARDEN_CACHE_TTL",
		Required:       false,
		DefaultValue:   "300",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	WardenVerifyCodeURL = EnvVariable{
		Name:           "WARDEN_VERIFY_CODE_URL",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	WardenOTPEnabled = EnvVariable{
		Name:           "WARDEN_OTP_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	WardenOTPSecretKey = EnvVariable{
		Name:           "WARDEN_OTP_SECRET_KEY",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}
)

func Initialize() error {
	// First, initialize language setting (before other validations that might use i18n)
	Language.Validate()
	lang := strings.ToLower(Language.Value)
	switch lang {
	case "zh":
		i18n.SetLanguage(i18n.LangZH)
	case "fr":
		i18n.SetLanguage(i18n.LangFR)
	case "it":
		i18n.SetLanguage(i18n.LangIT)
	case "ja":
		i18n.SetLanguage(i18n.LangJA)
	case "de":
		i18n.SetLanguage(i18n.LangDE)
	case "ko":
		i18n.SetLanguage(i18n.LangKO)
	default:
		i18n.SetLanguage(i18n.LangEN)
	}

	// Then validate all other configuration variables
	var envVariables = []*EnvVariable{&Debug, &AuthHost, &LoginPageTitle, &LoginPageFooterText, &Passwords, &UserHeaderName, &CookieDomain, &WardenURL, &WardenAPIKey, &WardenEnabled, &WardenCacheTTL, &WardenVerifyCodeURL, &WardenOTPEnabled, &WardenOTPSecretKey}

	for _, variable := range envVariables {
		err := variable.Validate()
		if err != nil {
			return err
		}

		// Only log non-empty configuration items
		if variable.Value != "" {
			logrus.Info("Config: ", variable.Name, " = ", variable.Value)
		}
	}

	// Log language setting
	if Language.Value != "" {
		logrus.Info("Config: ", Language.Name, " = ", Language.Value)
	}

	return nil
}
