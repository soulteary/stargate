package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	logger "github.com/soulteary/logger-kit/v3"

	"github.com/soulteary/stargate/src/internal/i18n"
)

// log is the package-level logger instance
var log *logger.Logger

// SessionExpiration is the session expiration time
const SessionExpiration = 24 * time.Hour

// Rate-limit defaults. They are the fallback used by handlers that run without
// Initialize (unit tests) and the source of the documented DefaultValue below,
// so the two can never drift apart.
const (
	DefaultRateLimitLoginMax        = 10
	DefaultRateLimitVerificationMax = 5
	DefaultRateLimitWindow          = time.Minute
)

// DefaultLoginUniformResendCooldown is the documented default of
// LOGIN_UNIFORM_RESEND_COOLDOWN and the fallback used by handlers that run
// without Initialize. It equals Herald's default RESEND_COOLDOWN.
const DefaultLoginUniformResendCooldown = 60 * time.Second

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

	LoginSupportURL = EnvVariable{Name: "LOGIN_SUPPORT_URL", PossibleValues: []string{"HTTPS URL or mailto address"}, Validator: ValidateLoginSupportURL}

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

	SessionExchangeRedirectPath = EnvVariable{
		Name:           "SESSION_EXCHANGE_REDIRECT_PATH",
		DefaultValue:   "/",
		PossibleValues: []string{"absolute same-origin path without query or fragment"},
		Validator:      ValidateSessionExchangeRedirectPath,
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

	// HeraldClientIPHeader names the request header that carries the end-user
	// address to Herald. Empty sends no header.
	HeraldClientIPHeader = EnvVariable{
		Name:           "HERALD_CLIENT_IP_HEADER",
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

	// Source freshness is opt-in until every Warden instance supports metadata v1.
	WardenSourceMaxAge = EnvVariable{
		Name: "WARDEN_SOURCE_MAX_AGE", DefaultValue: "0s",
		PossibleValues: []string{"0s (compatibility) or a positive duration"}, Validator: ValidateAny,
	}
	WardenSourceClockSkew = EnvVariable{
		Name: "WARDEN_SOURCE_CLOCK_SKEW", DefaultValue: "0s",
		PossibleValues: []string{"a non-negative duration smaller than WARDEN_SOURCE_MAX_AGE"}, Validator: ValidateAny,
	}
	MetricsAPIKey = EnvVariable{
		Name: "METRICS_API_KEY", DefaultValue: "", Sensitive: true,
		PossibleValues: []string{"at least 32 bytes; empty disables metrics access"}, Validator: ValidateAny,
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

	// Rate limiting. State is shared through the session Redis when
	// SESSION_STORAGE_ENABLED=true, so a multi-replica deployment enforces one
	// quota instead of one quota per replica. A limit of 0 disables that quota.
	RateLimitLoginMax = EnvVariable{
		Name:           "RATE_LIMIT_LOGIN_MAX",
		Required:       false,
		DefaultValue:   strconv.Itoa(DefaultRateLimitLoginMax),
		PossibleValues: []string{"0 (disabled) or a positive integer"},
		Validator:      ValidateAny,
	}

	RateLimitVerificationMax = EnvVariable{
		Name:           "RATE_LIMIT_VERIFICATION_MAX",
		Required:       false,
		DefaultValue:   strconv.Itoa(DefaultRateLimitVerificationMax),
		PossibleValues: []string{"0 (disabled) or a positive integer"},
		Validator:      ValidateAny,
	}

	RateLimitWindow = EnvVariable{
		Name:           "RATE_LIMIT_WINDOW",
		Required:       false,
		DefaultValue:   DefaultRateLimitWindow.String(),
		PossibleValues: []string{"1m", "30s", "5m"},
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
	// Opt-in: work-IM login requires an explicit trusted directory userid.
	LoginDingtalkRequireUserID = EnvVariable{
		Name:           "LOGIN_DINGTALK_REQUIRE_USERID",
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	// Opt-in: POST /_send_verify_code and Warden sign-in through POST /_login
	// answer with the same status, headers and body whether or not the
	// identifier is on the allow list. Response time is not equalized.
	LoginUniformResponses = EnvVariable{
		Name:           "LOGIN_UNIFORM_RESPONSES",
		DefaultValue:   "false",
		PossibleValues: []string{"true", "false"},
		Validator:      ValidateCaseInsensitivePossibleValues,
	}

	// LoginUniformResendCooldown is the per-identifier resend window that
	// uniform mode enforces before any lookup. It must be at least Herald's
	// RESEND_COOLDOWN.
	LoginUniformResendCooldown = EnvVariable{
		Name:           "LOGIN_UNIFORM_RESEND_COOLDOWN",
		DefaultValue:   "60s",
		PossibleValues: []string{"a duration of at least 1s"},
		Validator:      ValidateUniformResendCooldown,
	}
)

// warnUnattributableRateLimits reports the configuration in which rate limits
// stop being per-client. With an empty TRUSTED_PROXIES, forwarded headers are
// ignored and the client address is the direct peer. That is correct for a
// directly exposed deployment, but behind a reverse proxy every client is
// attributed to the proxy and therefore shares a single quota.
func warnUnattributableRateLimits() {
	if strings.TrimSpace(TrustedProxies.Value) != "" {
		return
	}
	if RateLimitLoginMax.ToInt(DefaultRateLimitLoginMax) <= 0 &&
		RateLimitVerificationMax.ToInt(DefaultRateLimitVerificationMax) <= 0 {
		return
	}
	log.Warn().Msg("TRUSTED_PROXIES is empty: the client address is the direct peer, so behind a reverse proxy every client shares one rate-limit quota. Set TRUSTED_PROXIES to the reverse-proxy source IPs or CIDRs.")
}

// PublishedDemoSessionExchangeSecret is the value shipped in the repository's
// docker-compose.yml, start-local.sh and deployment examples. It is public, so
// it must never protect an HTTPS deployment.
const PublishedDemoSessionExchangeSecret = "local-development-session-secret-change-me"

// PublishedDemoPasswords are the plaintext passwords shipped in the
// repository's docker-compose.yml and deployment examples. They are public, so
// they must never protect an HTTPS deployment.
var PublishedDemoPasswords = []string{"test1234", "test1337"}

// usesPublishedDemoPassword reports whether PASSWORDS lists one of the
// published demo passwords with the plaintext algorithm.
func usesPublishedDemoPassword() bool {
	algorithm, passwords, ok := ParsePasswords(Passwords.Value)
	if !ok || algorithm != "plaintext" {
		return false
	}
	for _, password := range passwords {
		for _, demo := range PublishedDemoPasswords {
			if password == demo {
				return true
			}
		}
	}
	return false
}

// warnPlaintextPasswords reports PASSWORDS stored with the plaintext algorithm
// in an HTTPS deployment. It stays a warning because existing deployments rely
// on it; bcrypt keeps the password out of the process environment.
func warnPlaintextPasswords() {
	if !CookieSecure.ToBool() {
		return
	}
	if algorithm, _, ok := ParsePasswords(Passwords.Value); ok && algorithm == "plaintext" {
		log.Warn().Msg("PASSWORDS uses the plaintext algorithm while COOKIE_SECURE is true: plaintext is intended for local development. Use bcrypt for production deployments.")
	}
}

// warnSharedPasswordStepUp reports a deployment in which step-up can still be
// passed with the shared password. Warden sessions must present a per-user
// proof, but a session created by the PASSWORDS login keeps that password as
// its only factor, so anyone who holds it can sign in and pass step-up.
func warnSharedPasswordStepUp() {
	if StepUpEnabled.ToBool() && WardenEnabled.ToBool() && HeraldEnabled.ToBool() && Passwords.Value != "" {
		log.Warn().Msg("STEP_UP_ENABLED with WARDEN_ENABLED and PASSWORDS: sessions created with the shared password still pass step-up with that password. Remove PASSWORDS to require a per-user proof for every session.")
	}
}

// warnUniformResponses reports the configurations in which
// LOGIN_UNIFORM_RESPONSES does less than an operator may expect.
func warnUniformResponses() {
	if !LoginUniformResponses.ToBool() {
		return
	}
	if !WardenEnabled.ToBool() || !HeraldEnabled.ToBool() {
		log.Warn().Msg("LOGIN_UNIFORM_RESPONSES is true but WARDEN_ENABLED or HERALD_ENABLED is false: uniform responses concern Warden sign-in through Herald, which this deployment does not offer. Requests with auth_method=warden are refused with the same error for every identifier.")
		return
	}
	if Debug.ToBool() && !CookieSecure.ToBool() {
		log.Warn().Msg("LOGIN_UNIFORM_RESPONSES is true while DEBUG is true and COOKIE_SECURE is false: debug_code is never returned in uniform mode.")
	}
	// Without the header Herald sees every user at Stargate's address. One
	// client can then use up the per-IP limit for all of them, and uniform
	// mode does not report the sends that Herald refuses.
	if strings.TrimSpace(HeraldClientIPHeader.Value) == "" {
		log.Warn().Msg("LOGIN_UNIFORM_RESPONSES is true and HERALD_CLIENT_IP_HEADER is empty: Herald counts RATE_LIMIT_PER_IP against Stargate's address for all users, and a send that Herald refuses is still shown as sent. Set HERALD_CLIENT_IP_HEADER together with HERALD_TRUSTED_PROXY_HEADER and HERALD_TRUSTED_PROXIES on Herald.")
	}
}

func Initialize(l *logger.Logger) error {
	log = l

	for _, validate := range []func() error{initializeLanguage, loadConfiguration, validatePasswordSettings, validateStepUpSettings, validateServiceSettings, validateHeaderAuthentication, validateStrictSettings} {
		if err := validate(); err != nil {
			return err
		}
	}

	// Log language setting
	if Language.Value != "" {
		log.Info().Str("name", Language.Name).Str("value", Language.Value).Msg("Config loaded")
	}

	warnUnattributableRateLimits()
	warnPlaintextPasswords()
	warnSharedPasswordStepUp()
	warnUniformResponses()

	// Initialize step-up matcher after configuration is loaded
	InitStepUpMatcher()

	return nil
}

func initializeLanguage() error {
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

	return nil
}

func loadConfiguration() error {
	for _, name := range []string{"WARDEN_OTP_ENABLED", "WARDEN_OTP_SECRET_KEY"} {
		if _, configured := os.LookupEnv(name); configured {
			return fmt.Errorf("%s has been removed; use Herald-backed TOTP with HERALD_TOTP_ENABLED", name)
		}
	}

	// Then validate all other configuration variables
	var envVariables = []*EnvVariable{&Debug, &AuthHost, &LoginPageTitle, &LoginPageFooterText, &LoginSupportURL, &Passwords, &PasswordHeaderAuthEnabled, &UserHeaderName, &TrustedProxies, &ProxyHeader, &CookieDomain, &CookieSecure, &CallbackAllowedHosts, &SessionExchangeSecret, &SessionExchangeRedirectPath, &Language, &Port, &WardenURL, &WardenAPIKey, &WardenHMACKeyID, &WardenHMACSecret, &WardenTLSCACertFile, &WardenTLSClientCert, &WardenTLSClientKey, &WardenTLSServerName, &WardenEnabled, &HeaderAuthEnabled, &HeaderAuthSharedSecret, &HeaderAuthSecretHeader, &WardenCacheTTL, &WardenSourceMaxAge, &WardenSourceClockSkew, &MetricsAPIKey, &HeraldURL, &HeraldAPIKey, &HeraldEnabled, &HeraldHMACSecret, &HeraldHMACKeyID, &HeraldClientIPHeader, &HeraldTLSCACertFile, &HeraldTLSClientCert, &HeraldTLSClientKey, &HeraldTLSServerName, &HeraldTOTPEnabled, &SessionStorageEnabled, &SessionStorageRedisAddr, &SessionStorageRedisPassword, &SessionStorageRedisDB, &SessionStorageRedisKeyPrefix, &AuditLogEnabled, &AuditLogFormat, &StepUpEnabled, &StepUpPaths, &OTLPEnabled, &OTLPEndpoint, &AuthRefreshEnabled, &AuthRefreshInterval, &RequestContextTimeout, &RateLimitLoginMax, &RateLimitVerificationMax, &RateLimitWindow, &LoginSMSEnabled, &LoginEmailEnabled, &LoginDingtalkRequireUserID, &LoginUniformResponses, &LoginUniformResendCooldown}

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

	return nil
}

func validatePasswordSettings() error {
	if SessionExchangeSecret.Value != "" && len(SessionExchangeSecret.Value) < 32 {
		return NewValidationError(SessionExchangeSecret.Name, "must contain at least 32 characters", SessionExchangeSecret.PossibleValues)
	}
	// The demo secret is published in this repository. It stays usable for the
	// local HTTP demo (COOKIE_SECURE=false) and is refused anywhere else.
	if CookieSecure.ToBool() && strings.TrimSpace(SessionExchangeSecret.Value) == PublishedDemoSessionExchangeSecret {
		return NewValidationError(SessionExchangeSecret.Name, "must not be the published demo value when COOKIE_SECURE is true; generate at least 32 random characters", SessionExchangeSecret.PossibleValues)
	}

	// The demo passwords are published like the demo secret. Other plaintext
	// passwords keep the startup warning below.
	if CookieSecure.ToBool() && usesPublishedDemoPassword() {
		return NewValidationError(Passwords.Name, "must not contain the published demo passwords when COOKIE_SECURE is true; choose your own and store them with bcrypt", Passwords.PossibleValues)
	}

	// PASSWORDS is required when not using Warden (password-only mode). When WardenEnabled=true, pure Warden deployment may omit PASSWORDS.
	if !WardenEnabled.ToBool() && Passwords.Value == "" {
		return NewValidationError(Passwords.Name, i18n.TStatic("error.config_required_not_set"), Passwords.PossibleValues)
	}
	if PasswordHeaderAuthEnabled.ToBool() && Passwords.Value == "" {
		return NewValidationError(PasswordHeaderAuthEnabled.Name, "requires PASSWORDS", PasswordHeaderAuthEnabled.PossibleValues)
	}
	// Step-up needs a proof for every session: Warden sessions use Herald,
	// shared-password sessions use PASSWORDS.
	if StepUpEnabled.ToBool() && Passwords.Value == "" && !HeraldEnabled.ToBool() {
		return NewValidationError(StepUpEnabled.Name, "requires PASSWORDS or HERALD_ENABLED=true for re-authentication", StepUpEnabled.PossibleValues)
	}
	return nil
}

func validateStepUpSettings() error {
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
	return nil
}

func validateServiceSettings() error {
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
	return validateHeraldSettings()
}

func validateHeaderAuthentication() error {
	if HeaderAuthEnabled.ToBool() {
		if !WardenEnabled.ToBool() {
			return NewValidationError(HeaderAuthEnabled.Name, "requires WARDEN_ENABLED=true", HeaderAuthEnabled.PossibleValues)
		}
		if strings.TrimSpace(HeaderAuthSharedSecret.Value) == "" {
			return NewValidationError(HeaderAuthSharedSecret.Name, i18n.TStatic("error.config_required_not_set"), HeaderAuthSharedSecret.PossibleValues)
		}
	}

	return nil
}

func validateHeraldSettings() error {
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
	return nil
}
