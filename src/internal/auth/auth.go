// Package auth provides authentication and session management functionality.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
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

	logrus.Debugf("Checking user in Warden list: phone=%s, mail=%s", phone, mail)
	result := client.CheckUserInList(ctx, phone, mail)
	if !result {
		logrus.Debugf("User not found in Warden list: phone=%s, mail=%s", phone, mail)
	}
	return result
}

// verifyCodeRequest represents the request body for verify code API
type verifyCodeRequest struct {
	Action string `json:"action"` // "send" or "verify"
	Phone  string `json:"phone,omitempty"`
	Mail   string `json:"mail,omitempty"`
	Code   string `json:"code,omitempty"` // Only used for verify action
}

// verifyCodeResponse represents the response from verify code API
type verifyCodeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SendVerifyCode sends a verification code to the user's phone or email via remote API.
//
// Parameters:
//   - ctx: Context for the request
//   - phone: User's phone number (optional)
//   - mail: User's email address (optional)
//
// Returns an error if the request fails or the API returns an error.
func SendVerifyCode(ctx context.Context, phone, mail string) error {
	verifyCodeURL := config.WardenVerifyCodeURL.String()
	if verifyCodeURL == "" {
		return fmt.Errorf("WARDEN_VERIFY_CODE_URL is not configured")
	}

	// Ensure we have at least one identifier
	if phone == "" && mail == "" {
		return fmt.Errorf("phone or mail must be provided")
	}

	// Ensure we have a valid context
	if ctx == nil {
		ctx = context.Background()
	} else {
		ctx = safeContext(ctx)
	}

	// Create request body
	reqBody := verifyCodeRequest{
		Action: "send",
		Phone:  phone,
		Mail:   mail,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", verifyCodeURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var apiResp verifyCodeResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response
	if resp.StatusCode != http.StatusOK || !apiResp.Success {
		errorMsg := apiResp.Error
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("API returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("verify code API error: %s", errorMsg)
	}

	logrus.Infof("Verification code sent successfully: phone=%s, mail=%s", phone, mail)
	return nil
}

// VerifyCode verifies a verification code with the remote API.
//
// Parameters:
//   - ctx: Context for the request
//   - phone: User's phone number (optional)
//   - mail: User's email address (optional)
//   - code: The verification code to verify
//
// Returns true if the code is valid, false otherwise.
// Returns an error if the request fails.
func VerifyCode(ctx context.Context, phone, mail, code string) (bool, error) {
	verifyCodeURL := config.WardenVerifyCodeURL.String()
	if verifyCodeURL == "" {
		return false, fmt.Errorf("WARDEN_VERIFY_CODE_URL is not configured")
	}

	// Ensure we have at least one identifier and a code
	if phone == "" && mail == "" {
		return false, fmt.Errorf("phone or mail must be provided")
	}
	if code == "" {
		return false, fmt.Errorf("code must be provided")
	}

	// Ensure we have a valid context
	if ctx == nil {
		ctx = context.Background()
	} else {
		ctx = safeContext(ctx)
	}

	// Create request body
	reqBody := verifyCodeRequest{
		Action: "verify",
		Phone:  phone,
		Mail:   mail,
		Code:   code,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", verifyCodeURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var apiResp verifyCodeResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check response
	if resp.StatusCode != http.StatusOK || !apiResp.Success {
		logrus.Debugf("Verification code verification failed: phone=%s, mail=%s, error=%s", phone, mail, apiResp.Error)
		return false, nil
	}

	logrus.Debugf("Verification code verified successfully: phone=%s, mail=%s", phone, mail)
	return true, nil
}

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
