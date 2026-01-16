package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/gofiber/fiber/v2/middleware/session"
)

// StateManager manages OAuth2 state parameters for CSRF protection
type StateManager struct{}

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
	storedState := sess.Get("oauth_state")
	if storedState == nil {
		return false
	}

	storedStateStr, ok := storedState.(string)
	if !ok {
		return false
	}

	// Clear the state after validation (single-use)
	sess.Delete("oauth_state")

	return storedStateStr == state
}

// SetState stores the state parameter in the session
func (sm *StateManager) SetState(sess *session.Session, state string) error {
	sess.Set("oauth_state", state)
	return sess.Save()
}

// GetUserInfoFromToken verifies the token and extracts user info
func (p *Provider) GetUserInfoFromToken(ctx context.Context, rawIDToken string) (*UserInfo, error) {
	idToken, err := p.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}

	return ExtractUserInfo(idToken)
}
