package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	logger "github.com/soulteary/logger-kit/v2"

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
		Sensitive:      true,
	}

	PasswordHeaderAuthEnabled = EnvVariable{
		Name:           "PASSWORD_HEADER_AUTH_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	UserHeaderName = EnvVariable{
		Name:           "USER_HEADER_NAME",
		Required:       false,
		DefaultValue:   "X-Forwarded-User",
		PossibleValues: []string{"*"},
		Validator:      ValidateNotEmptyString,
	}

	TrustedProxies = EnvVariable{
		Name:           "TRUSTED_PROXIES",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"127.0.0.1,10.0.0.0/8"},
		Validator:      ValidateIPOrCIDRList,
	}

	ProxyHeader = EnvVariable{
		Name:           "PROXY_HEADER",
		Required:       false,
		DefaultValue:   "X-Forwarded-For",
		PossibleValues: []string{"X-Forwarded-For", "CF-Connecting-IP"},
		Validator:      ValidateNotEmptyString,
	}

	CookieDomain = EnvVariable{
		Name:           "COOKIE_DOMAIN",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny, // Empty value is also valid (means not setting domain)
	}

	CookieSecure = EnvVariable{
		Name:           "COOKIE_SECURE",
		Required:       false,
		DefaultValue:   "true",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	CallbackAllowedHosts = EnvVariable{
		Name:           "CALLBACK_ALLOWED_HOSTS",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"app.example.com,admin.example.com"},
		Validator:      ValidateAny,
	}

	SessionExchangeSecret = EnvVariable{
		Name:           "SESSION_EXCHANGE_SECRET",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"at least 32 random characters"},
		Validator:      ValidateAny,
		Sensitive:      true,
	}

	Language = EnvVariable{
		Name:           "LANGUAGE",
		Required:       false,
		DefaultValue:   "en",
		PossibleValues: []string{"en", "zh", "fr", "it", "ja", "de", "ko"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	// Port is the service listening port (local development only). When empty, server uses default ":80".
	Port = EnvVariable{
		Name:           "PORT",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
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
		Sensitive:      true,
	}

	WardenHMACKeyID = EnvVariable{
		Name: "WARDEN_HMAC_KEY_ID", PossibleValues: []string{"*"}, Validator: ValidateAny,
	}
	WardenHMACSecret = EnvVariable{
		Name: "WARDEN_HMAC_SECRET", PossibleValues: []string{"*"}, Validator: ValidateAny, Sensitive: true,
	}
	WardenTLSCACertFile = EnvVariable{
		Name: "WARDEN_TLS_CA_CERT_FILE", PossibleValues: []string{"*"}, Validator: ValidateAny,
	}
	WardenTLSClientCert = EnvVariable{
		Name: "WARDEN_TLS_CLIENT_CERT_FILE", PossibleValues: []string{"*"}, Validator: ValidateAny,
	}
	WardenTLSClientKey = EnvVariable{
		Name: "WARDEN_TLS_CLIENT_KEY_FILE", PossibleValues: []string{"*"}, Validator: ValidateAny, Sensitive: true,
	}
	WardenTLSServerName = EnvVariable{
		Name: "WARDEN_TLS_SERVER_NAME", PossibleValues: []string{"*"}, Validator: ValidateAny,
	}

	WardenEnabled = EnvVariable{
		Name:           "WARDEN_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	HeaderAuthEnabled = EnvVariable{
		Name:           "HEADER_AUTH_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	HeaderAuthSharedSecret = EnvVariable{
		Name:           "HEADER_AUTH_SHARED_SECRET",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
		Sensitive:      true,
	}

	HeaderAuthSecretHeader = EnvVariable{
		Name:           "HEADER_AUTH_SECRET_HEADER",
		Required:       false,
		DefaultValue:   "X-Stargate-Header-Auth",
		PossibleValues: []string{"*"},
		Validator:      ValidateNotEmptyString,
	}

	WardenCacheTTL = EnvVariable{
		Name:           "WARDEN_CACHE_TTL",
		Required:       false,
		DefaultValue:   "300",
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
		Sensitive:      true,
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
		Sensitive:      true,
	}

	HeraldHMACKeyID = EnvVariable{
		Name:           "HERALD_HMAC_KEY_ID",
		Required:       false,
		DefaultValue:   "",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
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

	// Herald TOTP (per-user 2FA): when enabled, Stargate uses Herald client for TOTP (Herald proxies to herald-totp)
	HeraldTOTPEnabled = EnvVariable{
		Name:           "HERALD_TOTP_ENABLED",
		Required:       false,
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
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
		Sensitive:      true,
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
		DefaultValue:   "true",
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

	// RequestContextTimeout bounds request-scoped work, including Warden and
	// Herald calls. Fiber/fasthttp does not provide per-request cancellation on
	// ordinary client disconnects, so this deadline is the reliable cancellation
	// boundary propagated to upstream providers.
	RequestContextTimeout = EnvVariable{
		Name:           "REQUEST_CONTEXT_TIMEOUT",
		Required:       false,
		DefaultValue:   "10s",
		PossibleValues: []string{"*"},
		Validator:      ValidateAny,
	}

	// Login channel toggles: when false, SMS or email verification code login is disabled
	LoginSMSEnabled = EnvVariable{
		Name:           "LOGIN_SMS_ENABLED",
		Required:       false,
		DefaultValue:   "true",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	LoginEmailEnabled = EnvVariable{
		Name:           "LOGIN_EMAIL_ENABLED",
		Required:       false,
		DefaultValue:   "true",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
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

	for _, name := range []string{"WARDEN_OTP_ENABLED", "WARDEN_OTP_SECRET_KEY"} {
		if _, configured := os.LookupEnv(name); configured {
			return fmt.Errorf("%s has been removed; use Herald-backed TOTP with HERALD_TOTP_ENABLED", name)
		}
	}

	// Then validate all other configuration variables
	var envVariables = []*EnvVariable{&Debug, &AuthHost, &LoginPageTitle, &LoginPageFooterText, &Passwords, &PasswordHeaderAuthEnabled, &UserHeaderName, &TrustedProxies, &ProxyHeader, &CookieDomain, &CookieSecure, &CallbackAllowedHosts, &SessionExchangeSecret, &Language, &Port, &WardenURL, &WardenAPIKey, &WardenHMACKeyID, &WardenHMACSecret, &WardenTLSCACertFile, &WardenTLSClientCert, &WardenTLSClientKey, &WardenTLSServerName, &WardenEnabled, &HeaderAuthEnabled, &HeaderAuthSharedSecret, &HeaderAuthSecretHeader, &WardenCacheTTL, &HeraldURL, &HeraldAPIKey, &HeraldEnabled, &HeraldHMACSecret, &HeraldHMACKeyID, &HeraldTLSCACertFile, &HeraldTLSClientCert, &HeraldTLSClientKey, &HeraldTLSServerName, &HeraldTOTPEnabled, &SessionStorageEnabled, &SessionStorageRedisAddr, &SessionStorageRedisPassword, &SessionStorageRedisDB, &SessionStorageRedisKeyPrefix, &AuditLogEnabled, &AuditLogFormat, &StepUpEnabled, &StepUpPaths, &OTLPEnabled, &OTLPEndpoint, &AuthRefreshEnabled, &AuthRefreshInterval, &RequestContextTimeout, &LoginSMSEnabled, &LoginEmailEnabled}

	for _, variable := range envVariables {
		err := variable.Validate()
		if err != nil {
			return err
		}

		// Only log non-empty configuration items
		if variable.Value != "" {
			if variable.Sensitive {
				log.Info().Str("name", variable.Name).Bool("configured", true).Msg("Config loaded")
			} else {
				log.Info().Str("name", variable.Name).Str("value", variable.Value).Msg("Config loaded")
			}
		}
	}

	if SessionExchangeSecret.Value != "" && len(SessionExchangeSecret.Value) < 32 {
		return NewValidationError(SessionExchangeSecret.Name, "must contain at least 32 characters", SessionExchangeSecret.PossibleValues)
	}

	// PASSWORDS is required when not using Warden (password-only mode). When WardenEnabled=true, pure Warden deployment may omit PASSWORDS.
	if !WardenEnabled.ToBool() && Passwords.Value == "" {
		return NewValidationError(Passwords.Name, i18n.TStatic("error.config_required_not_set"), Passwords.PossibleValues)
	}
	if PasswordHeaderAuthEnabled.ToBool() && Passwords.Value == "" {
		return NewValidationError(PasswordHeaderAuthEnabled.Name, "requires PASSWORDS", PasswordHeaderAuthEnabled.PossibleValues)
	}
	if StepUpEnabled.ToBool() && Passwords.Value == "" {
		return NewValidationError(StepUpEnabled.Name, "requires PASSWORDS for re-authentication", StepUpEnabled.PossibleValues)
	}
	if StepUpEnabled.ToBool() && strings.TrimSpace(StepUpPaths.Value) == "" {
		return NewValidationError(StepUpPaths.Name, "must contain at least one protected path when step-up is enabled", StepUpPaths.PossibleValues)
	}
	if StepUpEnabled.ToBool() {
		usablePatterns := 0
		for _, pattern := range strings.Split(StepUpPaths.Value, ",") {
			if strings.TrimSpace(pattern) == "" {
				continue
			}
			usablePatterns++
			if !ValidStepUpPathPattern(pattern) {
				return NewValidationError(StepUpPaths.Name, "must contain only local absolute path patterns without control characters, fragments, backslashes, or dot segments", StepUpPaths.PossibleValues)
			}
		}
		if usablePatterns == 0 {
			return NewValidationError(StepUpPaths.Name, "must contain at least one usable protected path", StepUpPaths.PossibleValues)
		}
	}
	if StepUpEnabled.ToBool() && strings.TrimSpace(TrustedProxies.Value) == "" {
		return NewValidationError(TrustedProxies.Name, "must identify the proxies allowed to provide the original request URI when step-up is enabled", TrustedProxies.PossibleValues)
	}
	if WardenEnabled.ToBool() {
		if strings.TrimSpace(WardenURL.Value) == "" {
			return NewValidationError(WardenURL.Name, i18n.TStatic("error.config_required_not_set"), WardenURL.PossibleValues)
		}
	}
	if (WardenHMACKeyID.Value == "") != (WardenHMACSecret.Value == "") {
		return NewValidationError(WardenHMACSecret.Name, "HMAC key ID and secret must be configured together", WardenHMACSecret.PossibleValues)
	}
	if (WardenTLSClientCert.Value == "") != (WardenTLSClientKey.Value == "") {
		return NewValidationError(WardenTLSClientCert.Name, "client certificate and key must be configured together", WardenTLSClientCert.PossibleValues)
	}
	if HeraldEnabled.ToBool() && !WardenEnabled.ToBool() {
		return NewValidationError(HeraldEnabled.Name, "requires WARDEN_ENABLED=true to resolve authenticated users", HeraldEnabled.PossibleValues)
	}
	if HeraldEnabled.ToBool() {
		if strings.TrimSpace(HeraldURL.Value) == "" {
			return NewValidationError(HeraldURL.Name, i18n.TStatic("error.config_required_not_set"), HeraldURL.PossibleValues)
		}
	}
	if HeraldTOTPEnabled.ToBool() && !HeraldEnabled.ToBool() {
		return NewValidationError(HeraldTOTPEnabled.Name, "requires HERALD_ENABLED=true", HeraldTOTPEnabled.PossibleValues)
	}
	if WardenEnabled.ToBool() && Passwords.Value == "" && !HeraldEnabled.ToBool() && !HeaderAuthEnabled.ToBool() {
		return NewValidationError(WardenEnabled.Name, "requires PASSWORDS or an enabled Herald/header authentication path", WardenEnabled.PossibleValues)
	}
	if (HeraldTLSClientCert.Value == "") != (HeraldTLSClientKey.Value == "") {
		return NewValidationError(HeraldTLSClientCert.Name, "client certificate and key must be configured together", HeraldTLSClientCert.PossibleValues)
	}
	if HeaderAuthEnabled.ToBool() {
		if !WardenEnabled.ToBool() {
			return NewValidationError(HeaderAuthEnabled.Name, "requires WARDEN_ENABLED=true", HeaderAuthEnabled.PossibleValues)
		}
		if strings.TrimSpace(HeaderAuthSharedSecret.Value) == "" {
			return NewValidationError(HeaderAuthSharedSecret.Name, i18n.TStatic("error.config_required_not_set"), HeaderAuthSharedSecret.PossibleValues)
		}
	}

	if err := validateStrictSettings(); err != nil {
		return err
	}

	// Log language setting
	if Language.Value != "" {
		log.Info().Str("name", Language.Name).Str("value", Language.Value).Msg("Config loaded")
	}

	// Initialize step-up matcher after configuration is loaded
	InitStepUpMatcher()

	return nil
}
