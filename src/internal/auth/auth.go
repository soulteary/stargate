// Package auth provides authentication and session management functionality.
package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/soulteary/stargate/src/internal/config"
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

// GetUserID returns the user ID from the session.
//
// Parameters:
//   - session: The session to get the user ID from
//
// Returns the user ID if present, empty string otherwise.
func GetUserID(session *session.Session) string {
	if userID := session.Get("user_id"); userID != nil {
		if str, ok := userID.(string); ok {
			return str
		}
	}
	return ""
}

// GetEmail returns the email from the session.
//
// Parameters:
//   - session: The session to get the email from
//
// Returns the email if present, empty string otherwise.
func GetEmail(session *session.Session) string {
	if email := session.Get("email"); email != nil {
		if str, ok := email.(string); ok {
			return str
		}
	}
	return ""
}

// AuthenticateOIDC marks a session as authenticated with OIDC user info.
//
// Parameters:
//   - session: The session to authenticate
//   - userID: The user ID from the OIDC provider
//   - email: The email from the OIDC provider
//
// Returns an error if the session cannot be saved.
func AuthenticateOIDC(session *session.Session, userID, email string) error {
	session.Set("authenticated", true)
	session.Set("user_id", userID)
	session.Set("email", email)
	session.Set("provider", "oidc")
	return session.Save()
}

// GetForwardedUserValue returns the value to use for X-Forwarded-User header.
// Priority: user_id > email > "authenticated".
//
// Parameters:
//   - session: The session to get the user value from
//
// Returns the user ID, email, or "authenticated" as a fallback.
func GetForwardedUserValue(session *session.Session) string {
	if userID := GetUserID(session); userID != "" {
		return userID
	}
	if email := GetEmail(session); email != "" {
		return email
	}
	return "authenticated"
}
