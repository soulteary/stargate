package config

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
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

func validDNSName(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
				(char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

func parseHostPort(raw string) (string, error) {
	if raw == "" || raw != strings.TrimSpace(raw) || strings.ContainsAny(raw, "\\\r\n\t") {
		return "", strictValidationError(&AuthHost, "must be a host or host:port without scheme, path, query, or fragment")
	}
	parsed, err := url.Parse("//" + raw)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Hostname() == "" {
		return "", strictValidationError(&AuthHost, "must be a host or host:port without scheme, path, query, or fragment")
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return "", strictValidationError(&AuthHost, "must contain a TCP port between 1 and 65535")
		}
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if net.ParseIP(host) == nil && host != "localhost" && !validDNSName(host) {
		return "", strictValidationError(&AuthHost, "must contain a valid DNS name or IP address")
	}
	return host, nil
}

func validateAuthHostAndCookieDomain() error {
	authHost, err := parseHostPort(AuthHost.Value)
	if err != nil {
		return err
	}

	rawDomain := strings.TrimSpace(CookieDomain.Value)
	if rawDomain == "" {
		return nil
	}
	if rawDomain != CookieDomain.Value {
		return strictValidationError(&CookieDomain, "must not contain surrounding whitespace")
	}
	domain := strings.ToLower(strings.TrimPrefix(rawDomain, "."))
	if domain == "" || strings.Contains(rawDomain, ":") || net.ParseIP(domain) != nil || !validDNSName(domain) {
		return strictValidationError(&CookieDomain, "must be a registrable DNS domain without a port")
	}
	if _, err := publicsuffix.EffectiveTLDPlusOne(domain); err != nil {
		return strictValidationError(&CookieDomain, "must not be a public suffix")
	}
	if net.ParseIP(authHost) != nil || (authHost != domain && !strings.HasSuffix(authHost, "."+domain)) {
		return strictValidationError(&CookieDomain, "must equal AUTH_HOST or be its parent domain")
	}
	return nil
}

func isHTTPHeaderName(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", char) {
			continue
		}
		return false
	}
	return true
}

func validateHeaderContracts() error {
	headers := []*EnvVariable{&UserHeaderName, &ProxyHeader, &HeaderAuthSecretHeader}
	for _, variable := range headers {
		if !isHTTPHeaderName(variable.Value) {
			return strictValidationError(variable, "must be a valid HTTP header field name")
		}
	}

	forbidden := map[string]struct{}{
		"authorization": {}, "connection": {}, "content-length": {}, "cookie": {},
		"host": {}, "proxy-authorization": {}, "set-cookie": {}, "stargate-password": {},
		"te": {}, "trailer": {}, "transfer-encoding": {}, "upgrade": {},
		"x-forwarded-host": {}, "x-forwarded-proto": {}, "x-forwarded-uri": {},
		"x-user-mail": {}, "x-user-phone": {},
	}
	for _, variable := range headers {
		if _, blocked := forbidden[strings.ToLower(variable.Value)]; blocked {
			return strictValidationError(variable, "must not collide with a reserved authentication, routing, or hop-by-hop header")
		}
	}

	configured := []*EnvVariable{&UserHeaderName, &ProxyHeader, &HeaderAuthSecretHeader}
	for i, variable := range configured {
		for _, other := range configured[i+1:] {
			if strings.EqualFold(variable.Value, other.Value) {
				return strictValidationError(other, "must not reuse another configured header name")
			}
		}
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
		parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawPath != "" {
		return strictValidationError(variable, "must be an HTTP(S) service root URL without credentials, path, query, or fragment")
	}
	if _, err := parseHostPort(parsed.Host); err != nil {
		return strictValidationError(variable, "must contain a valid host and optional TCP port")
	}
	variable.Value = strings.TrimSuffix(variable.Value, "/")
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
			net.ParseIP(strings.TrimSuffix(parsed.Hostname(), ".")) != nil {
			return strictValidationError(variable, "must contain only comma-separated host[:port] values")
		}
	}
	return nil
}

func validateStrictSettings() error {
	if err := validateAuthHostAndCookieDomain(); err != nil {
		return err
	}
	if err := validateHeaderContracts(); err != nil {
		return err
	}
	if HeaderAuthEnabled.ToBool() && len(HeaderAuthSharedSecret.Value) < 32 {
		return strictValidationError(&HeaderAuthSharedSecret, "must contain at least 32 characters when trusted header authentication is enabled")
	}
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
