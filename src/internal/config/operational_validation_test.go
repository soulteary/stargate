package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthenticationConfigurationRejectsIncompleteTrust(t *testing.T) {
	variables := []*EnvVariable{&Passwords, &WardenEnabled, &WardenURL, &SessionExchangeSecret, &PasswordHeaderAuthEnabled, &StepUpEnabled, &HeaderAuthEnabled, &HeaderAuthSharedSecret, &HeraldEnabled, &HeraldURL, &HeraldTOTPEnabled, &WardenHMACKeyID, &WardenHMACSecret, &WardenTLSClientCert, &WardenTLSClientKey, &HeraldTLSClientCert, &HeraldTLSClientKey}
	previous := make([]string, len(variables))
	for i, v := range variables {
		previous[i] = v.Value
	}
	t.Cleanup(func() {
		for i, v := range variables {
			v.Value = previous[i]
		}
	})
	for _, tc := range []struct {
		name     string
		values   map[*EnvVariable]string
		validate func() error
		key      string
	}{
		{"short exchange secret", map[*EnvVariable]string{&SessionExchangeSecret: "short"}, validatePasswordSettings, SessionExchangeSecret.Name},
		{"password mode without password", nil, validatePasswordSettings, Passwords.Name},
		{"header password without password", map[*EnvVariable]string{&WardenEnabled: "true", &PasswordHeaderAuthEnabled: "true"}, validatePasswordSettings, PasswordHeaderAuthEnabled.Name},
		{"step-up without password", map[*EnvVariable]string{&WardenEnabled: "true", &StepUpEnabled: "true"}, validatePasswordSettings, StepUpEnabled.Name},
		{"Warden URL absent", map[*EnvVariable]string{&WardenEnabled: "true"}, validateServiceSettings, WardenURL.Name},
		{"unpaired Warden HMAC", map[*EnvVariable]string{&WardenHMACKeyID: "primary"}, validateServiceSettings, WardenHMACSecret.Name},
		{"unpaired Warden mTLS", map[*EnvVariable]string{&WardenTLSClientCert: "client.pem"}, validateServiceSettings, WardenTLSClientCert.Name},
		{"header auth without Warden", map[*EnvVariable]string{&HeaderAuthEnabled: "true"}, validateHeaderAuthentication, HeaderAuthEnabled.Name},
		{"header auth without secret", map[*EnvVariable]string{&HeaderAuthEnabled: "true", &WardenEnabled: "true"}, validateHeaderAuthentication, HeaderAuthSharedSecret.Name},
		{"Herald URL absent", map[*EnvVariable]string{&HeraldEnabled: "true", &WardenEnabled: "true"}, validateHeraldSettings, HeraldURL.Name},
		{"TOTP without Herald", map[*EnvVariable]string{&HeraldTOTPEnabled: "true"}, validateHeraldSettings, HeraldTOTPEnabled.Name},
		{"unpaired Herald mTLS", map[*EnvVariable]string{&HeraldTLSClientKey: "client.key"}, validateHeraldSettings, HeraldTLSClientCert.Name},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range variables {
				v.Value = ""
			}
			for v, value := range tc.values {
				v.Value = value
			}
			err := tc.validate()
			require.Error(t, err)
			require.Equal(t, tc.key, err.(*ValidationError).KeyName)
		})
	}
}

func TestDNSLabelsRejectMalformedNames(t *testing.T) {
	for _, host := range []string{"", strings.Repeat("a", 254), "-bad.example", "bad-.example", strings.Repeat("a", 64) + ".example", "bad..example", "bad_name.example", "例.example"} {
		require.False(t, validDNSName(host), host)
	}
	require.True(t, validDNSName("auth-1.example.com"))
	_, err := parseHostPort("https://auth.example.com/path")
	require.Error(t, err)
}

func TestIntegerConversionUsesFallbackOnlyForInvalidValues(t *testing.T) {
	v := EnvVariable{Value: "invalid"}
	require.Equal(t, 42, v.ToInt(42))
	v.Value = " 0 "
	require.Equal(t, 0, v.ToInt(42))
}

func TestStepUpMatcherInitializesLazilyAndMatchesRoot(t *testing.T) {
	oldMatcher, oldEnabled, oldPaths := stepUpMatcher, StepUpEnabled.Value, StepUpPaths.Value
	t.Cleanup(func() { stepUpMatcher = oldMatcher; StepUpEnabled.Value = oldEnabled; StepUpPaths.Value = oldPaths })
	stepUpMatcher = nil
	StepUpEnabled.Value = "true"
	StepUpPaths.Value = " , /, "
	require.True(t, GetStepUpMatcher().RequiresStepUp("/private"))
	StepUpPaths.Value = ""
	InitStepUpMatcher()
	require.False(t, GetStepUpMatcher().RequiresStepUp("/private"))
}

func TestStrictValidationRejectsMalformedOperationalAddresses(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func()
	}{
		{"malformed host", func() { AuthHost.Value = "auth.example:bad" }},
		{"invalid DNS host", func() { AuthHost.Value = "bad_name.example" }},
		{"cookie whitespace", func() { CookieDomain.Value = " example.com" }},
		{"Herald invalid host", func() { HeraldURL.Value = "https://bad_name.example" }},
		{"Redis unavailable address", func() { SessionStorageEnabled.Value = "true"; SessionStorageRedisAddr.Value = "" }},
		{"Herald invalid URL", func() { HeraldURL.Value = "https://host:bad" }},
	} {
		t.Run(tc.name, func(t *testing.T) { setValidStrictTestValues(t); tc.set(); require.Error(t, validateStrictSettings()) })
	}
	v := EnvVariable{Name: "DURATION", Value: "invalid"}
	require.Error(t, validatePositiveDuration(&v))
}
