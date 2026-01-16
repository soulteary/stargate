package config

import (
	"fmt"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// IsOIDCEnabled returns true if OIDC authentication is enabled
func IsOIDCEnabled() bool {
	return OIDCEnabled.String() == "true"
}

// GetOIDCProviderName returns the configured OIDC provider name
func GetOIDCProviderName() string {
	if OIDCProviderName.Value != "" {
		return OIDCProviderName.Value
	}
	return "OIDC"
}

// ValidateOIDCConfig validates that all required OIDC fields are set when OIDC is enabled
func ValidateOIDCConfig() error {
	if !IsOIDCEnabled() {
		return nil
	}

	// Validate required fields
	if OIDCIssuerURL.Value == "" {
		return fmt.Errorf(i18n.T("error.config_required_not_set"), OIDCIssuerURL.Name)
	}
	if OIDCClientID.Value == "" {
		return fmt.Errorf(i18n.T("error.config_required_not_set"), OIDCClientID.Name)
	}
	if OIDCClientSecret.Value == "" {
		return fmt.Errorf(i18n.T("error.config_required_not_set"), OIDCClientSecret.Name)
	}

	return nil
}
