// Package auth provides authentication and session management functionality.
package auth

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	secure "github.com/soulteary/secure-kit"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
)

// SessionCookieName is the name of the session cookie used for authentication.
const SessionCookieName = "stargate_session_id"

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
	// Schema: "algorithm:pass1|pass2|pass3"
	passwordsRaw := config.Passwords.String()
	if passwordsRaw == "" {
		return "", []string{}
	}

	parts := strings.SplitN(passwordsRaw, ":", 2)
	if len(parts) < 2 {
		// Invalid format, return empty to fail safely
		return "", []string{}
	}

	algorithm := parts[0]
	passwordsStr := parts[1]
	if passwordsStr == "" {
		return algorithm, []string{}
	}

	passwords := strings.Split(passwordsStr, "|")
	for k, v := range passwords {
		normalized := strings.ToUpper(strings.TrimSpace(v))
		normalized = strings.ReplaceAll(normalized, " ", "")
		passwords[k] = normalized
	}
	return algorithm, passwords
}

// CheckPassword validates a password against the configured valid passwords.
// It normalizes the input password (uppercase, trim spaces) and checks it against
// all configured passwords using the configured algorithm.
//
// Parameters:
//   - password: The password to check
//
// Returns true if the password matches any of the configured passwords, false otherwise.
func CheckPassword(password string) bool {
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

	tryToCheck := strings.ToUpper(strings.TrimSpace(password))
	tryToCheck = strings.ReplaceAll(tryToCheck, " ", "")

	for _, validPassword := range validPasswords {
		if algorithmResolver.Check(validPassword, tryToCheck) {
			return true
		}
	}

	return false
}

// Authenticate marks a session as authenticated by setting the "authenticated" flag.
//
// Parameters:
//   - session: The session to authenticate
//
// Returns an error if the session cannot be saved.
func Authenticate(session *session.Session) error {
	session.Set("authenticated", true)
	return session.Save()
}

// Unauthenticate destroys a session, effectively logging out the user.
//
// Parameters:
//   - session: The session to destroy
//
// Returns an error if the session cannot be destroyed.
func Unauthenticate(session *session.Session) error {
	return session.Destroy()
}

// IsAuthenticated checks if a session is authenticated.
//
// Parameters:
//   - session: The session to check
//
// Returns true if the session has the "authenticated" flag set, false otherwise.
func IsAuthenticated(session *session.Session) bool {
	return session.Get("authenticated") != nil
}

// wardenClient is a global instance of the Warden client.
// It's initialized once and reused for all requests.
var wardenClient *warden.Client
var wardenClientInit sync.Once

// ResetWardenClientForTesting resets the Warden client and initialization state for testing purposes.
// This function should only be used in tests.
func ResetWardenClientForTesting() {
	wardenClient = nil
	wardenClientInit = sync.Once{}
}

// InitWardenClient initializes the Warden client if enabled.
// This should be called after configuration is loaded.
func InitWardenClient() {
	wardenClientInit.Do(func() {
		if !config.WardenEnabled.ToBool() {
			logrus.Debug("Warden is not enabled, skipping client initialization")
			return
		}

		wardenURL := config.WardenURL.String()
		if wardenURL == "" {
			logrus.Warn("WARDEN_URL is not set, Warden client will not be initialized")
			return
		}

		// Parse cache TTL
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
			WithCacheTTL(cacheTTL).
			WithLogger(warden.NewLogrusAdapter(logrus.StandardLogger()))

		// Create client
		client, err := warden.NewClient(opts)
		if err != nil {
			logrus.Warnf("Failed to initialize Warden client: %v. Check WARDEN_URL and WARDEN_ENABLED configuration.", err)
			return
		}

		wardenClient = client
		logrus.Info("Warden client initialized successfully")
	})
}

// getWardenClient returns the warden client, initializing it if necessary.
func getWardenClient() *warden.Client {
	// Try to initialize if not already done
	InitWardenClient()
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

// CheckUserInList checks if a user (by phone or mail) is in the Warden allow list.
//
// Parameters:
//   - ctx: Context for the request (can be nil, will use background context)
//   - phone: User's phone number (optional, can be empty)
//   - mail: User's email address (optional, can be empty)
//
// Returns true if the user is in the allow list, false otherwise.
// If Warden is not enabled or client is not initialized, returns false.
func CheckUserInList(ctx context.Context, phone, mail string) bool {
	if !config.WardenEnabled.ToBool() {
		logrus.Debug("Warden is not enabled, skipping user list check")
		return false
	}

	client := getWardenClient()
	if client == nil {
		logrus.Warn("Warden client is not initialized, cannot check user in list. Make sure WARDEN_URL is set and WARDEN_ENABLED is true.")
		return false
	}

	// Ensure we have a valid context
	// If ctx is nil, use background context
	if ctx == nil {
		ctx = context.Background()
	} else {
		// In test environments, fasthttp.RequestCtx may have an invalid context
		// that causes panics when used. Use a safe wrapper that falls back to
		// background context if the original context is invalid.
		ctx = safeContext(ctx)
	}

	// Normalize input (trim spaces, lowercase mail)
	phone = strings.TrimSpace(phone)
	mail = strings.TrimSpace(strings.ToLower(mail))

	if phone == "" && mail == "" {
		logrus.Debug("CheckUserInList called with both phone and mail empty")
		return false
	}

	logrus.Debugf("Checking user in Warden list: phone=%s, mail=%s", secure.MaskPhone(phone), secure.MaskEmail(mail))
	exists := client.CheckUserInList(ctx, phone, mail)
	if exists {
		logrus.Debugf("User found and active: phone=%s, mail=%s", secure.MaskPhone(phone), secure.MaskEmail(mail))
	} else {
		logrus.Debugf("User not found in Warden list or not active: phone=%s, mail=%s", secure.MaskPhone(phone), secure.MaskEmail(mail))
	}
	return exists
}

// GetUserInfo fetches complete user information from Warden by phone or mail.
//
// Returns the user information if found and active, nil otherwise.
// If Warden is not enabled or client is not initialized, returns nil.
func GetUserInfo(ctx context.Context, phone, mail string) *warden.AllowListUser {
	if !config.WardenEnabled.ToBool() {
		logrus.Debug("Warden is not enabled, skipping user info fetch")
		return nil
	}

	client := getWardenClient()
	if client == nil {
		logrus.Warn("Warden client is not initialized, cannot get user info. Make sure WARDEN_URL is set and WARDEN_ENABLED is true.")
		return nil
	}

	// Ensure we have a valid context
	if ctx == nil {
		ctx = context.Background()
	} else {
		ctx = safeContext(ctx)
	}

	// Normalize input (trim spaces, lowercase mail)
	phone = strings.TrimSpace(phone)
	mail = strings.TrimSpace(strings.ToLower(mail))

	if phone == "" && mail == "" {
		logrus.Debug("GetUserInfo called with both phone and mail empty")
		return nil
	}

	logrus.Debugf("Fetching user info from Warden: phone=%s, mail=%s", secure.MaskPhone(phone), secure.MaskEmail(mail))

	var (
		user *warden.AllowListUser
		err  error
	)

	if phone != "" {
		user, err = client.GetUserByIdentifier(ctx, phone, "", "")
		if err != nil {
			if sdkErr, ok := err.(*warden.Error); ok && sdkErr.Code == warden.ErrCodeNotFound && mail != "" {
				logrus.Debugf("User not found by phone, falling back to mail: phone=%s, mail=%s", secure.MaskPhone(phone), secure.MaskEmail(mail))
				user, err = client.GetUserByIdentifier(ctx, "", mail, "")
			} else {
				logrus.Debugf("Failed to get user info from Warden: %v (phone=%s, mail=%s)", err, secure.MaskPhone(phone), secure.MaskEmail(mail))
				return nil
			}
		}
	} else {
		user, err = client.GetUserByIdentifier(ctx, "", mail, "")
	}

	if err != nil {
		logrus.Debugf("Failed to get user info from Warden: %v (phone=%s, mail=%s)", err, secure.MaskPhone(phone), secure.MaskEmail(mail))
		return nil
	}

	if user == nil {
		logrus.Debugf("User not found in Warden: phone=%s, mail=%s", secure.MaskPhone(phone), secure.MaskEmail(mail))
		return nil
	}

	// Check if user is active
	if !user.IsActive() {
		logrus.Warnf("User status is not active: phone=%s, mail=%s, status=%s", secure.MaskPhone(phone), secure.MaskEmail(mail), user.Status)
		return nil
	}

	logrus.Debugf("Fetched user info from Warden: user_id=%s, phone=%s, mail=%s, status=%s", user.UserID, secure.MaskPhone(user.Phone), secure.MaskEmail(user.Mail), user.Status)
	return user
}

// Note: SendVerifyCode and VerifyCode functions have been removed.
// Verification code functionality is now handled by the Herald service.

// VerifyOTP verifies a TOTP (Time-based One-Time Password) code.
//
// Parameters:
//   - secret: The OTP secret key (base32 encoded)
//   - code: The OTP code to verify (6 digits)
//
// Returns true if the code is valid, false otherwise.
func VerifyOTP(secret, code string) bool {
	if secret == "" {
		logrus.Debug("OTP secret is empty, cannot verify")
		return false
	}

	if code == "" {
		logrus.Debug("OTP code is empty, cannot verify")
		return false
	}

	// Validate code is 6 digits
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		logrus.Debugf("OTP code length is invalid: expected 6, got %d", len(code))
		return false
	}

	// Verify TOTP code
	// Using 30 second window, allowing 1 time step skew (previous/current/next window)
	valid := totp.Validate(code, secret)
	if !valid {
		logrus.Debug("OTP code verification failed")
		return false
	}

	logrus.Debug("OTP code verified successfully")
	return true
}

// GetOTPSecret returns the OTP secret key from configuration.
// This can be extended to fetch from remote API if needed.
func GetOTPSecret() string {
	return config.WardenOTPSecretKey.String()
}
