package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func setValidStrictTestValues(t *testing.T) {
	t.Helper()
	variables := []*EnvVariable{
		&CookieDomain, &CallbackAllowedHosts, &SessionExchangeSecret, &Port,
		&WardenCacheTTL, &AuthRefreshInterval, &SessionStorageRedisDB,
		&SessionStorageEnabled, &SessionStorageRedisAddr, &WardenURL, &HeraldURL,
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
		name     string
		configure func()
		key      string
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
