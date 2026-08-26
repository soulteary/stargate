package config

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func strictValidationError(variable *EnvVariable, reason string) error {
	return NewValidationError(variable.Name, reason, variable.PossibleValues)
}

func validatePositiveInteger(variable *EnvVariable) error {
	value, err := strconv.Atoi(strings.TrimSpace(variable.Value))
	if err != nil || value <= 0 {
		return strictValidationError(variable, "must be a positive integer")
	}
	return nil
}

func validateHTTPServiceURL(variable *EnvVariable) error {
	if strings.TrimSpace(variable.Value) == "" {
		return nil
	}
	parsed, err := url.Parse(variable.Value)
	if err != nil || parsed.Host == "" || parsed.User != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return strictValidationError(variable, "must be an HTTP(S) service URL without credentials, query, or fragment")
	}
	return nil
}

func validateCallbackHostList(variable *EnvVariable) error {
	for _, raw := range strings.Split(variable.Value, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		parsed, err := url.Parse("//" + raw)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" ||
			parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Hostname() == "" ||
			net.ParseIP(parsed.Hostname()) != nil {
			return strictValidationError(variable, "must contain only comma-separated host[:port] values")
		}
	}
	return nil
}

func validateStrictSettings() error {
	if strings.TrimSpace(CookieDomain.Value) != "" || strings.TrimSpace(CallbackAllowedHosts.Value) != "" {
		if len(SessionExchangeSecret.Value) < 32 {
			return strictValidationError(&SessionExchangeSecret, "must contain at least 32 characters when cross-domain callbacks are enabled")
		}
	}
	if err := validateCallbackHostList(&CallbackAllowedHosts); err != nil {
		return err
	}
	if Port.Value != "" {
		port, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(Port.Value), ":"))
		if err != nil || port < 1 || port > 65535 {
			return strictValidationError(&Port, "must be a TCP port between 1 and 65535")
		}
	}
	if err := validatePositiveInteger(&WardenCacheTTL); err != nil {
		return err
	}
	if _, err := time.ParseDuration(AuthRefreshInterval.Value); err != nil || AuthRefreshInterval.ToDuration() <= 0 {
		return strictValidationError(&AuthRefreshInterval, "must be a positive Go duration")
	}
	redisDBValue := SessionStorageRedisDB.Value
	redisDB, err := strconv.Atoi(redisDBValue)
	if err != nil || redisDB < 0 || strings.TrimSpace(redisDBValue) != redisDBValue {
		return strictValidationError(&SessionStorageRedisDB, "must be a non-negative integer")
	}
	if SessionStorageEnabled.ToBool() && strings.TrimSpace(SessionStorageRedisAddr.Value) == "" {
		return strictValidationError(&SessionStorageRedisAddr, "must be set when Redis session storage is enabled")
	}
	if err := validateHTTPServiceURL(&WardenURL); err != nil {
		return err
	}
	if err := validateHTTPServiceURL(&HeraldURL); err != nil {
		return err
	}
	return nil
}
