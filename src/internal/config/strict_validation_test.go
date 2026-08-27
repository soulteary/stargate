package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func setValidStrictTestValues(t *testing.T) {
	t.Helper()
	variables := []*EnvVariable{
		&AuthHost, &CookieDomain, &CallbackAllowedHosts, &SessionExchangeSecret, &Port,
		&WardenCacheTTL, &AuthRefreshInterval, &SessionStorageRedisDB,
		&SessionStorageEnabled, &SessionStorageRedisAddr, &WardenURL, &HeraldURL,
		&UserHeaderName, &ProxyHeader, &HeaderAuthEnabled, &HeaderAuthSharedSecret, &HeaderAuthSecretHeader,
	}
	values := make([]string, len(variables))
	for i, variable := range variables {
		values[i] = variable.Value
	}
	t.Cleanup(func() {
		for i, variable := range variables {
			variable.Value = values[i]
		}
	})

	AuthHost.Value = "auth.example.com"
	CookieDomain.Value = ""
	CallbackAllowedHosts.Value = ""
	SessionExchangeSecret.Value = ""
	Port.Value = ""
	WardenCacheTTL.Value = "300"
	AuthRefreshInterval.Value = "5m"
	SessionStorageRedisDB.Value = "0"
	SessionStorageEnabled.Value = "false"
	SessionStorageRedisAddr.Value = "localhost:6379"
	WardenURL.Value = ""
	HeraldURL.Value = ""
	UserHeaderName.Value = "X-Forwarded-User"
	ProxyHeader.Value = "X-Forwarded-For"
	HeaderAuthEnabled.Value = "false"
	HeaderAuthSharedSecret.Value = ""
	HeaderAuthSecretHeader.Value = "X-Stargate-Header-Auth"
}

func TestValidateStrictSettingsAcceptsOperationalDefaults(t *testing.T) {
	setValidStrictTestValues(t)
	testza.AssertNoError(t, validateStrictSettings())
}

func TestValidateStrictSettingsRequiresCrossDomainExchangeSecret(t *testing.T) {
	setValidStrictTestValues(t)
	CallbackAllowedHosts.Value = "app.example.com"
	SessionExchangeSecret.Value = "too-short"

	err := validateStrictSettings()
	testza.AssertNotNil(t, err)
	testza.AssertEqual(t, SessionExchangeSecret.Name, err.(*ValidationError).KeyName)
}

func TestValidateStrictSettingsRejectsInvalidNumbers(t *testing.T) {
	tests := []struct {
		name      string
		configure func()
		key       string
	}{
		{"port", func() { Port.Value = "70000" }, Port.Name},
		{"cache ttl", func() { WardenCacheTTL.Value = "0" }, WardenCacheTTL.Name},
		{"redis db", func() { SessionStorageRedisDB.Value = "-1" }, SessionStorageRedisDB.Name},
		{"refresh duration", func() { AuthRefreshInterval.Value = "soon" }, AuthRefreshInterval.Name},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValidStrictTestValues(t)
			test.configure()
			err := validateStrictSettings()
			testza.AssertNotNil(t, err)
			testza.AssertEqual(t, test.key, err.(*ValidationError).KeyName)
		})
	}
}

func TestValidateStrictSettingsRejectsUnsafeServiceURL(t *testing.T) {
	setValidStrictTestValues(t)
	WardenURL.Value = "https://user:password@warden.example.com?token=secret"

	err := validateStrictSettings()
	testza.AssertNotNil(t, err)
	testza.AssertEqual(t, WardenURL.Name, err.(*ValidationError).KeyName)
}

func TestValidateStrictSettingsNormalizesServiceRootURL(t *testing.T) {
	setValidStrictTestValues(t)
	WardenURL.Value = "https://warden.example.com/"
	HeraldURL.Value = "https://herald.example.com/"

	testza.AssertNoError(t, validateStrictSettings())
	testza.AssertEqual(t, "https://warden.example.com", WardenURL.Value)
	testza.AssertEqual(t, "https://herald.example.com", HeraldURL.Value)
}

func TestValidateStrictSettingsRejectsServiceURLPath(t *testing.T) {
	setValidStrictTestValues(t)
	WardenURL.Value = "https://warden.example.com/api"

	err := validateStrictSettings()
	testza.AssertNotNil(t, err)
	testza.AssertEqual(t, WardenURL.Name, err.(*ValidationError).KeyName)
}

func TestValidateStrictSettingsRejectsInvalidAuthHost(t *testing.T) {
	for _, host := range []string{"https://auth.example.com", "auth.example.com/login", "user@auth.example.com", "auth.example.com:70000"} {
		t.Run(host, func(t *testing.T) {
			setValidStrictTestValues(t)
			AuthHost.Value = host

			err := validateStrictSettings()
			testza.AssertNotNil(t, err)
			testza.AssertEqual(t, AuthHost.Name, err.(*ValidationError).KeyName)
		})
	}
}

func TestValidateStrictSettingsChecksCookieDomainBoundary(t *testing.T) {
	tests := []struct {
		domain string
		valid  bool
	}{
		{domain: ".example.com", valid: true},
		{domain: "auth.example.com", valid: true},
		{domain: "example.net", valid: false},
		{domain: "com", valid: false},
		{domain: "127.0.0.1", valid: false},
	}
	for _, test := range tests {
		t.Run(test.domain, func(t *testing.T) {
			setValidStrictTestValues(t)
			CookieDomain.Value = test.domain
			SessionExchangeSecret.Value = "0123456789abcdef0123456789abcdef"

			err := validateStrictSettings()
			if test.valid {
				testza.AssertNoError(t, err)
				return
			}
			testza.AssertNotNil(t, err)
			testza.AssertEqual(t, CookieDomain.Name, err.(*ValidationError).KeyName)
		})
	}
}

func TestValidateStrictSettingsRejectsHeaderCollisions(t *testing.T) {
	tests := []struct {
		configure func()
		key       string
	}{
		{configure: func() { HeaderAuthSecretHeader.Value = "X-Forwarded-Uri" }, key: HeaderAuthSecretHeader.Name},
		{configure: func() { HeaderAuthSecretHeader.Value = "invalid header" }, key: HeaderAuthSecretHeader.Name},
		{configure: func() { UserHeaderName.Value = "Authorization" }, key: UserHeaderName.Name},
		{configure: func() { ProxyHeader.Value = UserHeaderName.Value }, key: ProxyHeader.Name},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			setValidStrictTestValues(t)
			test.configure()

			err := validateStrictSettings()
			testza.AssertNotNil(t, err)
			testza.AssertEqual(t, test.key, err.(*ValidationError).KeyName)
		})
	}
}

func TestValidateStrictSettingsRequiresStrongHeaderAuthSecret(t *testing.T) {
	setValidStrictTestValues(t)
	HeaderAuthEnabled.Value = "true"
	HeaderAuthSharedSecret.Value = "too-short"

	err := validateStrictSettings()
	testza.AssertNotNil(t, err)
	testza.AssertEqual(t, HeaderAuthSharedSecret.Name, err.(*ValidationError).KeyName)
}

func TestValidateStrictSettingsRejectsPaddedRedisDB(t *testing.T) {
	setValidStrictTestValues(t)
	SessionStorageRedisDB.Value = " 2 "

	err := validateStrictSettings()
	testza.AssertNotNil(t, err)
	testza.AssertEqual(t, SessionStorageRedisDB.Name, err.(*ValidationError).KeyName)
}

func TestValidateStrictSettingsRejectsIPCallbackHost(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "127.0.0.1."} {
		t.Run(host, func(t *testing.T) {
			setValidStrictTestValues(t)
			CallbackAllowedHosts.Value = host
			SessionExchangeSecret.Value = "0123456789abcdef0123456789abcdef"

			err := validateStrictSettings()
			testza.AssertNotNil(t, err)
			testza.AssertEqual(t, CallbackAllowedHosts.Name, err.(*ValidationError).KeyName)
		})
	}
}
