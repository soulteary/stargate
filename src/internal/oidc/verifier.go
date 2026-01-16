package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/gofiber/fiber/v2/middleware/session"
)

// StateManager manages OAuth2 state parameters for CSRF protection
type StateManager struct{}

const (
	oauthStateKey    = "oauth_state"
	oauthCallbackKey = "oauth_callback"
)

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{}
}

// GenerateState generates a cryptographically random state parameter
func (sm *StateManager) GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateState validates the state parameter from the callback
func (sm *StateManager) ValidateState(sess *session.Session, state string) bool {
	storedState := sess.Get(oauthStateKey)
	if storedState == nil {
		return false
	}

	storedStateStr, ok := storedState.(string)
	if !ok {
		return false
	}

	// Clear the state after validation (single-use)
	sess.Delete(oauthStateKey)

	return storedStateStr == state
}

// SetState stores the state parameter in the session
func (sm *StateManager) SetState(sess *session.Session, state string) error {
	sess.Set(oauthStateKey, state)
	return sess.Save()
}

// SetStateWithCallback stores state and callback in the session and saves it once.
func (sm *StateManager) SetStateWithCallback(sess *session.Session, state, callback string) error {
	sess.Set(oauthStateKey, state)
	if callback == "" {
		sess.Delete(oauthCallbackKey)
	} else {
		sess.Set(oauthCallbackKey, callback)
	}
	return sess.Save()
}

// GetCallback returns the stored callback host, if any.
func (sm *StateManager) GetCallback(sess *session.Session) string {
	value := sess.Get(oauthCallbackKey)
	if value == nil {
		return ""
	}
	if stored, ok := value.(string); ok {
		return stored
	}
	return ""
}

// ClearCallback removes the stored callback from the session.
func (sm *StateManager) ClearCallback(sess *session.Session) {
	sess.Delete(oauthCallbackKey)
}

// GetUserInfoFromToken verifies the token and extracts user info
func (p *Provider) GetUserInfoFromToken(ctx context.Context, rawIDToken string) (*UserInfo, error) {
	idToken, err := p.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}

	return ExtractUserInfo(idToken)
}
