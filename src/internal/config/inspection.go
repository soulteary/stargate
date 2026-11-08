package config

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
