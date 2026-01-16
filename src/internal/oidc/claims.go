package oidc

import (
	"errors"

	oidcext "github.com/coreos/go-oidc/v3/oidc"
)

// UserInfo represents the user information extracted from OIDC claims
type UserInfo struct {
	UserID string
	Email  string
}

// ExtractUserInfo extracts user information from ID token claims
func ExtractUserInfo(token *oidcext.IDToken) (*UserInfo, error) {
	var claims struct {
		UserID string `json:"sub"`
		Email  string `json:"email"`
	}

	if err := token.Claims(&claims); err != nil {
		return nil, errors.New("failed to extract claims from token")
	}

	if claims.UserID == "" {
		return nil, errors.New("missing sub claim in token")
	}

	return &UserInfo{
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}
