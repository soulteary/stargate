package config

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/soulteary/stargate/src/internal/i18n"
)

// SessionExpiration 会话过期时间
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
		Validator:      ValidateAny, // 空值也是有效的（表示不设置域名）
	}

	Language = EnvVariable{
		Name:           "LANGUAGE",
		Required:       false,
		DefaultValue:   "en",
		PossibleValues: []string{"en", "zh"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}
)

func Initialize() error {
	// First, initialize language setting (before other validations that might use i18n)
	Language.Validate()
	lang := strings.ToLower(Language.Value)
	if lang == "zh" {
		i18n.SetLanguage(i18n.LangZH)
	} else {
		i18n.SetLanguage(i18n.LangEN)
	}

	// Then validate all other configuration variables
	var envVariables = []*EnvVariable{&Debug, &AuthHost, &LoginPageTitle, &LoginPageFooterText, &Passwords, &UserHeaderName, &CookieDomain}

	for _, variable := range envVariables {
		err := variable.Validate()
		if err != nil {
			return err
		}

		// 只记录非空值的配置项
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
