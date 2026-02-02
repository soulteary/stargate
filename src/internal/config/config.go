package config

import (
	"strings"
	"time"

	logger "github.com/soulteary/logger-kit"

	"github.com/soulteary/stargate/src/internal/i18n"
)

// log is the package-level logger instance
var log *logger.Logger

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
		Required:       false, // Required only when WardenEnabled=false; see Initialize()
		DefaultValue:   "",
		PossibleValues: []string{"algorithm:pass1|pass2|pass3"},
		Validator:      ValidatePasswordsOrEmpty,
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

	// WardenVerifyCodeURL has been removed - verification codes are now handled by Herald service

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

	HeraldURL = EnvVariable{
		Name:           "HERALD_URL",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	HeraldAPIKey = EnvVariable{
		Name:           "HERALD_API_KEY",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	HeraldEnabled = EnvVariable{
		Name:           "HERALD_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	HeraldHMACSecret = EnvVariable{
		Name:           "HERALD_HMAC_SECRET",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny, // Empty value is also valid (means using API key instead)
	}

	HeraldTLSCACertFile = EnvVariable{
		Name:           "HERALD_TLS_CA_CERT_FILE",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	HeraldTLSClientCert = EnvVariable{
		Name:           "HERALD_TLS_CLIENT_CERT_FILE",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	HeraldTLSClientKey = EnvVariable{
		Name:           "HERALD_TLS_CLIENT_KEY_FILE",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	HeraldTLSServerName = EnvVariable{
		Name:           "HERALD_TLS_SERVER_NAME",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	// Herald TOTP (per-user 2FA): when enabled, Stargate calls herald-totp for Status/Verify
	HeraldTOTPBaseURL = EnvVariable{
		Name:           "HERALD_TOTP_BASE_URL",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}
	HeraldTOTPEnabled = EnvVariable{
		Name:           "HERALD_TOTP_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}
	HeraldTOTPAPIKey = EnvVariable{
		Name:           "HERALD_TOTP_API_KEY",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}
	HeraldTOTPHMACSecret = EnvVariable{
		Name:           "HERALD_TOTP_HMAC_SECRET",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	SessionStorageEnabled = EnvVariable{
		Name:           "SESSION_STORAGE_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	SessionStorageRedisAddr = EnvVariable{
		Name:           "SESSION_STORAGE_REDIS_ADDR",
		Required:       false,
		DefaultValue:   "localhost:6379",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	SessionStorageRedisPassword = EnvVariable{
		Name:           "SESSION_STORAGE_REDIS_PASSWORD",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	SessionStorageRedisDB = EnvVariable{
		Name:           "SESSION_STORAGE_REDIS_DB",
		Required:       false,
		DefaultValue:   "0",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	SessionStorageRedisKeyPrefix = EnvVariable{
		Name:           "SESSION_STORAGE_REDIS_KEY_PREFIX",
		Required:       false,
		DefaultValue:   "stargate:session:",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	AuditLogEnabled = EnvVariable{
		Name:           "AUDIT_LOG_ENABLED",
		Required:       false,
		DefaultValue:   "true",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	AuditLogFormat = EnvVariable{
		Name:           "AUDIT_LOG_FORMAT",
		Required:       false,
		DefaultValue:   "json",
		PossibleValues: []string{"json", "text"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	StepUpEnabled = EnvVariable{
		Name:           "STEP_UP_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	StepUpPaths = EnvVariable{
		Name:           "STEP_UP_PATHS",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	// OpenTelemetry config
	OTLPEnabled = EnvVariable{
		Name:           "OTLP_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	OTLPEndpoint = EnvVariable{
		Name:           "OTLP_ENDPOINT",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	// Auth refresh config
	AuthRefreshEnabled = EnvVariable{
		Name:           "AUTH_REFRESH_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	AuthRefreshInterval = EnvVariable{
		Name:           "AUTH_REFRESH_INTERVAL",
		Required:       false,
		DefaultValue:   "5m",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}
)

func Initialize(l *logger.Logger) error {
	log = l

	// First, initialize language setting (before other validations that might use i18n)
	if err := Language.Validate(); err != nil {
		return err
	}
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
	var envVariables = []*EnvVariable{&Debug, &AuthHost, &LoginPageTitle, &LoginPageFooterText, &Passwords, &UserHeaderName, &CookieDomain, &WardenURL, &WardenAPIKey, &WardenEnabled, &WardenCacheTTL, &WardenOTPEnabled, &WardenOTPSecretKey, &HeraldURL, &HeraldAPIKey, &HeraldEnabled, &HeraldHMACSecret, &HeraldTLSCACertFile, &HeraldTLSClientCert, &HeraldTLSClientKey, &HeraldTLSServerName, &HeraldTOTPBaseURL, &HeraldTOTPEnabled, &HeraldTOTPAPIKey, &HeraldTOTPHMACSecret, &SessionStorageEnabled, &SessionStorageRedisAddr, &SessionStorageRedisPassword, &SessionStorageRedisDB, &SessionStorageRedisKeyPrefix, &AuditLogEnabled, &AuditLogFormat, &StepUpEnabled, &StepUpPaths, &OTLPEnabled, &OTLPEndpoint, &AuthRefreshEnabled, &AuthRefreshInterval}

	for _, variable := range envVariables {
		err := variable.Validate()
		if err != nil {
			return err
		}

		// Only log non-empty configuration items
		if variable.Value != "" {
			log.Info().Str("name", variable.Name).Str("value", variable.Value).Msg("Config loaded")
		}
	}

	// PASSWORDS is required when not using Warden (password-only mode). When WardenEnabled=true, pure Warden deployment may omit PASSWORDS.
	if !WardenEnabled.ToBool() && Passwords.Value == "" {
		return NewValidationError(Passwords.Name, i18n.TStatic("error.config_required_not_set"), Passwords.PossibleValues)
	}

	// Log language setting
	if Language.Value != "" {
		log.Info().Str("name", Language.Name).Str("value", Language.Value).Msg("Config loaded")
	}

	// Initialize step-up matcher after configuration is loaded
	InitStepUpMatcher()

	return nil
}
