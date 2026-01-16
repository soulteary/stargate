package config

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
