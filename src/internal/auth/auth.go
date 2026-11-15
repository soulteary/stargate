// Package auth provides authentication and session management functionality.
package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	logger "github.com/soulteary/logger-kit/v3"
	session "github.com/soulteary/session-kit/v3"
	"github.com/soulteary/stargate/src/internal/config"
	internal_tls "github.com/soulteary/stargate/src/internal/tlsconfig"
	"github.com/soulteary/warden/pkg/warden"
)

// log is the package-level logger instance
var log *logger.Logger

// SessionCookieName is the name of the session cookie used for authentication.
// Note: This constant is kept for backward compatibility. Consider using session-kit's Config.CookieName.
const SessionCookieName = "stargate_session_id"

// Re-export session-kit functions for session management.
// This provides a unified interface for session operations in Stargate.
var (
	// Authenticate marks a fiber session as authenticated.
	Authenticate = session.Authenticate
	// Unauthenticate destroys a fiber session.
	Unauthenticate = session.Unauthenticate
	// IsAuthenticated checks if a fiber session is authenticated.
	IsAuthenticated = session.IsAuthenticated
	// SetUserID sets the user ID in a fiber session.
	SetUserID = session.SetUserID
	// GetUserID gets the user ID from a fiber session.
	GetUserID = session.GetUserID
	// SetEmail sets the email in a fiber session.
	SetEmail = session.SetEmail
	// GetEmail gets the email from a fiber session.
	GetEmail = session.GetEmail
	// SetPhone sets the phone in a fiber session.
	SetPhone = session.SetPhone
	// GetPhone gets the phone from a fiber session.
	GetPhone = session.GetPhone
	// SetAMR sets the authentication methods references in a fiber session.
	SetAMR = session.SetAMR
	// GetAMR gets the authentication methods references from a fiber session.
	GetAMR = session.GetAMR
	// AddAMR adds an authentication method reference to a fiber session.
	AddAMR = session.AddAMR
	// HasAMR checks if a fiber session has a specific authentication method.
	HasAMR = session.HasAMR
	// SetScopes sets the authorization scopes in a fiber session.
	SetScopes = session.SetScopes
	// GetScopes gets the authorization scopes from a fiber session.
	GetScopes = session.GetScopes
	// HasScope checks if a fiber session has a specific scope.
	HasScope = session.HasScope
	// UpdateLastAccess updates the last access timestamp in a fiber session.
	UpdateLastAccess = session.UpdateLastAccess
)

// GetValidPasswords parses the password configuration and returns the algorithm and list of valid passwords.
// The configuration format is: "algorithm:pass1|pass2|pass3"
//
// Returns:
//   - algorithm: The password hashing algorithm (e.g., "plaintext", "bcrypt")
//   - passwords: A slice of valid password hashes/values
//
// Note: This function assumes the password format has been validated during configuration initialization.
// If the format is invalid, it will return empty values, which will cause authentication to fail safely.
func GetValidPasswords() (string, []string) {
	algorithm, passwords, ok := config.ParsePasswords(config.Passwords.String())
	if !ok {
		return "", []string{}
	}
	return algorithm, passwords
}

// CheckPassword validates a password against the configured valid passwords.
// Passwords are opaque, case-sensitive byte strings. No whitespace or case
// normalization is performed before verification.
//
// Parameters:
//   - password: The password to check
//
// Returns true if the password matches any of the configured passwords, false otherwise.
func CheckPassword(password string) bool {
	if password == "" {
		return false
	}
	algo, validPasswords := GetValidPasswords()

	// If no valid passwords configured, authentication fails
	if algo == "" || len(validPasswords) == 0 {
		return false
	}

	// Check if algorithm is supported
	algorithmResolver, exists := config.SupportedAlgorithms[algo]
	if !exists {
		return false
	}

	for _, validPassword := range validPasswords {
		if algorithmResolver.Check(validPassword, password) {
			return true
		}
	}

	return false
}

// wardenClient is a global instance of the Warden client.
// It's initialized once and reused for all requests.
var wardenClient *warden.Client
var wardenClientInit sync.Once
var wardenClientInitErr error

// ResetWardenClientForTesting resets the Warden client and initialization state for testing purposes.
// This function should only be used in tests.
func ResetWardenClientForTesting() {
	wardenClient = nil
	wardenClientInit = sync.Once{}
	wardenClientInitErr = nil
}

// InitWardenClient initializes the Warden client if enabled.
// This should be called after configuration is loaded.
func InitWardenClient(l *logger.Logger) error {
	log = l
	wardenClientInit.Do(func() {
		if !config.WardenEnabled.ToBool() {
			log.Debug().Msg("Warden is not enabled, skipping client initialization")
			return
		}

		wardenURL := config.WardenURL.String()
		if wardenURL == "" {
			wardenClientInitErr = fmt.Errorf("WARDEN_URL is required when Warden is enabled")
			return
		}

		// The TTL only governs the SDK's user-list cache. Stargate looks users
		// up one identifier at a time, which the SDK does not cache, so the
		// value changes nothing here; it is passed on for compatibility.
		cacheTTL := 300 * time.Second // Default 5 minutes
		if ttlStr := config.WardenCacheTTL.String(); ttlStr != "" {
			if parsedTTL, err := strconv.Atoi(ttlStr); err == nil && parsedTTL > 0 {
				cacheTTL = time.Duration(parsedTTL) * time.Second
			}
		}

		// Create SDK options
		opts := warden.DefaultOptions().
			WithBaseURL(wardenURL).
			WithAPIKey(config.WardenAPIKey.String()).
			WithCacheTTL(cacheTTL)

		if keyID, secret := config.WardenHMACKeyID.String(), config.WardenHMACSecret.String(); keyID != "" || secret != "" {
			opts = opts.WithHMAC(keyID, secret)
		}
		// An audience selects signature version v3, which the Warden server
		// accepts only when the name is listed in its WARDEN_HMAC_AUDIENCES.
		if audience := config.WardenHMACAudience.String(); audience != "" {
			opts = opts.WithHMACAudience(audience)
		}

		tlsConfig, err := buildWardenTLSConfig(
			config.WardenTLSCACertFile.String(), config.WardenTLSClientCert.String(),
			config.WardenTLSClientKey.String(), config.WardenTLSServerName.String(),
		)
		if err != nil {
			wardenClientInitErr = fmt.Errorf("configure Warden TLS: %w", err)
			return
		}
		if config.WardenSourceMaxAge.ToDuration() > 0 {
			base := http.DefaultTransport.(*http.Transport).Clone()
			base.TLSClientConfig = tlsConfig
			wrapped := &sourceFreshnessTransport{base: base, origin: wardenURL, maxAge: config.WardenSourceMaxAge.ToDuration(), skew: config.WardenSourceClockSkew.ToDuration(), now: time.Now}
			// The SDK accepts a concrete Transport. Registered protocols let us check
			// response metadata without changing its signing or decoding behavior.
			transport := &http.Transport{}
			transport.RegisterProtocol("http", wrapped)
			transport.RegisterProtocol("https", wrapped)
			opts = opts.WithTransport(transport)
		} else if tlsConfig != nil {
			opts = opts.WithTLSConfig(tlsConfig)
		}

		// Create client
		client, err := warden.NewClient(opts)
		if err != nil {
			wardenClientInitErr = fmt.Errorf("create Warden client: %w", err)
			return
		}

		wardenClient = client
		log.Info().Msg("Warden client initialized successfully")
	})

	return wardenClientInitErr
}

func buildWardenTLSConfig(caFile, certFile, keyFile, serverName string) (*tls.Config, error) {
	return internal_tls.Build("Warden", caFile, certFile, keyFile, serverName)
}

// getWardenClient returns the warden client.
// Note: InitWardenClient must be called with a logger before this function is used.
func getWardenClient() *warden.Client {
	return wardenClient
}

// safeContext returns a safe context wrapper that prevents panics from invalid contexts.
// If the original context is invalid (e.g., uninitialized fasthttp.RequestCtx in tests),
// it returns context.Background() instead.
func safeContext(ctx context.Context) context.Context {
	// Use a closure to capture the result
	var safeCtx context.Context
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Context is invalid, use background context
				safeCtx = context.Background()
			}
		}()

		// Try to access Done() to check if context is valid
		// If this panics, the defer will catch it and set safeCtx to background
		_ = ctx.Done()

		// Context appears valid, use it
		safeCtx = ctx
	}()

	// If safeCtx is still nil (shouldn't happen, but be safe), use background
	if safeCtx == nil {
		return context.Background()
	}

	return safeCtx
}

// NormalizePhone 规范化手机号：去除首尾空白及中间空格、连字符等，便于与 Warden 比对。
// 系统自动填充可能产生 "138 0013 8000" 这类带空格的格式，统一规范为纯数字再参与查询与校验。
func NormalizePhone(phone string) string {
	s := strings.TrimSpace(phone)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u00a0", "") // NBSP
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "－", "") // 全角连字符
	return s
}

// IsValidPhone 在规范化后校验是否为有效号码：仅包含数字且长度在合理范围内（如 10～15 位）。
func IsValidPhone(phone string) bool {
	s := NormalizePhone(phone)
	if s == "" {
		return false
	}
	if len(s) < 10 || len(s) > 15 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ErrUserUnavailable distinguishes a confirmed missing/disabled identity from
// dependency failures. Only the former invalidates an authenticated session.
var ErrUserUnavailable = errors.New("warden identity is missing or inactive")

// ErrWardenCredentialsRejected reports that Warden answered 401 to Stargate's
// own request: the API key or the HMAC signature was refused. It says nothing
// about the looked-up user and is never ErrUserUnavailable, so callers fail
// closed without treating the identity as missing or revoked.
var ErrWardenCredentialsRejected = errors.New("warden rejected the credentials of this Stargate instance")

// wardenCredentialKind names the credential the client sends, the way the SDK
// selects it: an HMAC signature when a key ID or secret is set, otherwise the
// API key. Only the kind is logged, never a value.
func wardenCredentialKind() string {
	switch {
	case config.WardenHMACKeyID.String() != "" || config.WardenHMACSecret.String() != "":
		return "hmac"
	case config.WardenAPIKey.String() != "":
		return "api_key"
	default:
		return "none"
	}
}

// logWardenCredentialsRejected writes the operator-facing line for a 401 from
// Warden. It carries no secret, key ID, audience name or user identifier.
func logWardenCredentialsRejected() {
	if log == nil {
		return
	}
	log.Error().
		Str("credential", wardenCredentialKind()).
		Bool("hmac_audience_set", config.WardenHMACAudience.String() != "").
		Msg("Warden rejected this Stargate instance's own credentials with HTTP 401; the request failed before any allow-list lookup, so this is not a missing user. " +
			"Check WARDEN_API_KEY, or WARDEN_HMAC_KEY_ID, WARDEN_HMAC_SECRET and the clocks of both hosts; " +
			"when WARDEN_HMAC_AUDIENCE is set, Warden must be 1.8.0 or newer and list the same name in WARDEN_HMAC_AUDIENCES; " +
			"a Warden with WARDEN_HMAC_REQUIRE_AUDIENCE=true rejects requests signed without an audience")
}

// ErrWardenAccessDenied reports that Warden answered 403 to Stargate's own
// request: Warden accepted or did not check the credential and refused this
// client. Like ErrWardenCredentialsRejected it says nothing about the
// looked-up user and is never ErrUserUnavailable.
var ErrWardenAccessDenied = errors.New("warden denied access to this Stargate instance")

// logWardenAccessDenied writes the operator-facing line for a 403 from Warden.
// It carries no secret, key ID, address or user identifier.
func logWardenAccessDenied() {
	if log == nil {
		return
	}
	log.Error().
		Str("credential", wardenCredentialKind()).
		Msg("Warden refused this Stargate instance with HTTP 403; the request was refused before any allow-list lookup, so this is not a missing user. " +
			"Warden answers 403 when the address it sees for Stargate is not in its IP_WHITELIST (TRUSTED_PROXY_IPS decides whether a forwarded address is used), " +
			"or when its WARDEN_SERVICE_POLICIES does not grant the action directory:lookup to the key ID in WARDEN_HMAC_KEY_ID. " +
			"A proxy between Stargate and Warden can also answer 403")
}

// WardenLookupError is a lookup that produced no answer about the user for a
// reason other than a 401 or 403 from Warden. Class names the reason and
// holds no request or response data.
type WardenLookupError struct {
	Class string
}

func (e *WardenLookupError) Error() string {
	return "warden lookup failed: " + e.Class
}

// Failure classes of a Warden lookup, as returned by WardenLookupFailureClass.
const (
	WardenFailureCredentialsRejected = "credentials_rejected" // Warden answered 401
	WardenFailureAccessDenied        = "access_denied"        // Warden answered 403
	WardenFailureRateLimited         = "rate_limited"         // Warden answered 429
	WardenFailureServerError         = "server_error"         // Warden answered 500, 502 or 503
	WardenFailureUnexpectedStatus    = "unexpected_status"    // any other non-200 status
	WardenFailureNoResponse          = "no_response"          // connection, TLS or timeout failure
	WardenFailureInvalidResponse     = "invalid_response"     // the body could not be used
	WardenFailureSourceFreshness     = "source_freshness"     // WARDEN_SOURCE_MAX_AGE evidence missing or expired
	WardenFailureClientUnavailable   = "client_unavailable"   // Warden disabled or client not initialized
	WardenFailureUnknown             = "unknown"
)

// WardenLookupFailureClass names why LookupUserInfo returned no answer about
// the user. It returns "" for a nil error and for ErrUserUnavailable, which is
// an answer: the user is missing or inactive.
func WardenLookupFailureClass(err error) string {
	var lookupErr *WardenLookupError
	switch {
	case err == nil || errors.Is(err, ErrUserUnavailable):
		return ""
	case errors.Is(err, ErrWardenCredentialsRejected):
		return WardenFailureCredentialsRejected
	case errors.Is(err, ErrWardenAccessDenied):
		return WardenFailureAccessDenied
	case errors.As(err, &lookupErr):
		return lookupErr.Class
	default:
		return WardenFailureUnknown
	}
}

// classifyWardenFailure maps an SDK error that is neither 401, 403 nor 404 to
// a failure class.
func classifyWardenFailure(err error) string {
	if errors.Is(err, errSourceEvidence) {
		return WardenFailureSourceFreshness
	}
	var sdkErr *warden.Error
	if !errors.As(err, &sdkErr) {
		return WardenFailureUnknown
	}
	switch sdkErr.Code {
	case warden.ErrCodeRateLimited:
		return WardenFailureRateLimited
	case warden.ErrCodeServerError:
		return WardenFailureServerError
	case warden.ErrCodeInvalidResponse:
		return WardenFailureInvalidResponse
	case warden.ErrCodeSourceFreshness:
		return WardenFailureSourceFreshness
	case warden.ErrCodeRequestFailed:
		if sdkErr.StatusCode != 0 {
			return WardenFailureUnexpectedStatus
		}
		return WardenFailureNoResponse
	default:
		return WardenFailureUnknown
	}
}

// GetUserInfo returns an active directory identity or nil. Call LookupUserInfo
// when dependency failures must be distinguished from revoked identities.
func GetUserInfo(ctx context.Context, phone, mail string) *warden.AllowListUser {
	user, _, _ := LookupUserInfo(ctx, phone, mail)
	return user
}

// LookupUserInfo returns a directory identity and, when configured, the exact
// source deadline checked for that lookup. Failures never extend a deadline.
//
// ErrUserUnavailable is the only error that is an answer about the user. Every
// other error means the lookup itself failed; WardenLookupFailureClass names
// the reason.
func LookupUserInfo(ctx context.Context, phone, mail string) (*warden.AllowListUser, time.Time, error) {
	if !config.WardenEnabled.ToBool() || getWardenClient() == nil {
		return nil, time.Time{}, &WardenLookupError{Class: WardenFailureClientUnavailable}
	}
	if ctx == nil {
		ctx = context.Background()
	} else {
		ctx = safeContext(ctx)
	}
	evidence := &sourceEvidence{}
	ctx = context.WithValue(ctx, sourceEvidenceKey{}, evidence)
	phone = NormalizePhone(phone)
	mail = strings.TrimSpace(strings.ToLower(mail))
	if phone == "" && mail == "" {
		return nil, time.Time{}, ErrUserUnavailable
	}
	client := getWardenClient()
	lookup := func(phone, mail string) (*warden.AllowListUser, error) {
		user, err := client.GetUserByIdentifier(ctx, phone, mail, "")
		// The SDK consumes the body before returning. Evidence can expire while
		// reading it, including a 404 body; check before classifying revocation.
		if config.WardenSourceMaxAge.ToDuration() > 0 && evidence.expired() {
			return nil, errSourceEvidence
		}
		return user, err
	}
	var user *warden.AllowListUser
	var err error
	if phone != "" {
		user, err = lookup(phone, "")
		var sdkErr *warden.Error
		if errors.As(err, &sdkErr) && sdkErr.Code == warden.ErrCodeNotFound && mail != "" {
			user, err = lookup("", mail)
		}
	} else {
		user, err = lookup("", mail)
	}
	if err != nil {
		var sdkErr *warden.Error
		if errors.As(err, &sdkErr) && sdkErr.Code == warden.ErrCodeNotFound {
			return nil, time.Time{}, ErrUserUnavailable
		}
		if errors.As(err, &sdkErr) && sdkErr.Code == warden.ErrCodeUnauthorized {
			logWardenCredentialsRejected()
			return nil, time.Time{}, ErrWardenCredentialsRejected
		}
		// The SDK reports a 403 with its generic request-failed code; the
		// status tells it apart.
		if errors.As(err, &sdkErr) && sdkErr.StatusCode == http.StatusForbidden {
			logWardenAccessDenied()
			return nil, time.Time{}, ErrWardenAccessDenied
		}
		return nil, time.Time{}, &WardenLookupError{Class: classifyWardenFailure(err)}
	}
	if user == nil || !user.IsActive() {
		return nil, time.Time{}, ErrUserUnavailable
	}
	deadline := evidence.deadline()
	if config.WardenSourceMaxAge.ToDuration() > 0 && evidence.expired() {
		return nil, time.Time{}, &WardenLookupError{Class: WardenFailureSourceFreshness}
	}
	return user, deadline, nil
}
