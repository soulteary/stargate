package config

import "strings"

// ValidateServiceRoot checks the startup service URL contract without changing configuration.
func ValidateServiceRoot(raw string) error {
	v := EnvVariable{Name: "service URL", Value: raw}
	return validateHTTPServiceURL(&v)
}

// ValidateCallbackHosts checks the startup callback list syntax without changing configuration.
func ValidateCallbackHosts(raw string) error {
	v := EnvVariable{Name: "CALLBACK_ALLOWED_HOSTS", Value: raw}
	return validateCallbackHostList(&v)
}

// ValidateCrossDomainExchange checks the startup shared-secret requirement without changing configuration.
func ValidateCrossDomainExchange(cookieDomain, callbacks, secret string) error {
	if strings.TrimSpace(cookieDomain) != "" || strings.TrimSpace(callbacks) != "" {
		if len(secret) < 32 {
			return strictValidationError(&SessionExchangeSecret, "must contain at least 32 characters when cross-domain callbacks are enabled")
		}
	}
	return nil
}
