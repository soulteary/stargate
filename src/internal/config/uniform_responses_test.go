package config

import (
	"bytes"
	"testing"
	"time"

	logger "github.com/soulteary/logger-kit/v3"
	"github.com/stretchr/testify/require"
)

func setUniformTestEnv(t *testing.T, values map[string]string) {
	t.Helper()
	isolateGlobalState(t)
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	for _, name := range []string{"WARDEN_ENABLED", "WARDEN_URL", "HERALD_ENABLED", "HERALD_URL", "HERALD_CLIENT_IP_HEADER", "DEBUG", "COOKIE_SECURE", "LOGIN_UNIFORM_RESPONSES", "LOGIN_UNIFORM_RESEND_COOLDOWN"} {
		t.Setenv(name, values[name])
	}
}

func TestLoginUniformResponsesDefaultsToOff(t *testing.T) {
	for _, tc := range []struct {
		value   string
		want    bool
		invalid bool
	}{
		{value: "", want: false},
		{value: "false", want: false},
		{value: "true", want: true},
		{value: "TRUE", want: true},
		{value: "1", invalid: true},
		{value: "yes", invalid: true},
	} {
		t.Run("LOGIN_UNIFORM_RESPONSES="+tc.value, func(t *testing.T) {
			setUniformTestEnv(t, map[string]string{"LOGIN_UNIFORM_RESPONSES": tc.value})
			err := Initialize(testLogger())
			if tc.invalid {
				require.Error(t, err)
				require.Contains(t, err.Error(), "LOGIN_UNIFORM_RESPONSES")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, LoginUniformResponses.ToBool())
		})
	}
}

func TestLoginUniformResendCooldownIsValidatedAtStartup(t *testing.T) {
	for _, tc := range []struct {
		value   string
		want    time.Duration
		invalid bool
	}{
		{value: "", want: DefaultLoginUniformResendCooldown},
		{value: "60s", want: time.Minute},
		{value: "1s", want: time.Second},
		{value: "1500ms", want: 1500 * time.Millisecond},
		{value: "2m", want: 2 * time.Minute},
		{value: "999ms", invalid: true},
		{value: "0s", invalid: true},
		{value: "-60s", invalid: true},
		{value: "60", invalid: true},
		{value: "soon", invalid: true},
	} {
		t.Run("LOGIN_UNIFORM_RESEND_COOLDOWN="+tc.value, func(t *testing.T) {
			// The cooldown is validated whether or not the mode is on, so a
			// typo is found before the flag is turned on.
			setUniformTestEnv(t, map[string]string{"LOGIN_UNIFORM_RESEND_COOLDOWN": tc.value})
			err := Initialize(testLogger())
			if tc.invalid {
				require.Error(t, err)
				require.Contains(t, err.Error(), "LOGIN_UNIFORM_RESEND_COOLDOWN")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, LoginUniformResendCooldown.ToDuration())
		})
	}
}

// The documented default and the fallback constant must not drift apart.
func TestLoginUniformResendCooldownDefaultMatchesConstant(t *testing.T) {
	parsed, err := time.ParseDuration(LoginUniformResendCooldown.DefaultValue)
	require.NoError(t, err)
	require.Equal(t, DefaultLoginUniformResendCooldown, parsed)
	require.True(t, ValidateUniformResendCooldown(EnvVariable{Value: LoginUniformResendCooldown.DefaultValue}))
}

func TestInitializeWarnsAboutUniformResponses(t *testing.T) {
	const (
		notOffered  = "uniform responses concern Warden sign-in through Herald, which this deployment does not offer. Requests with auth_method=warden are refused with the same error for every identifier."
		debugCode   = "debug_code is never returned in uniform mode"
		sharedLimit = "Herald counts RATE_LIMIT_PER_IP against Stargate's address for all users, and a send that Herald refuses is still shown as sent. Set HERALD_CLIENT_IP_HEADER together with HERALD_TRUSTED_PROXY_HEADER and HERALD_TRUSTED_PROXIES on Herald."
	)
	warden := map[string]string{"WARDEN_ENABLED": "true", "WARDEN_URL": "http://warden.test"}
	herald := map[string]string{"HERALD_ENABLED": "true", "HERALD_URL": "http://herald.test"}
	merge := func(parts ...map[string]string) map[string]string {
		merged := map[string]string{}
		for _, part := range parts {
			for name, value := range part {
				merged[name] = value
			}
		}
		return merged
	}
	on := map[string]string{"LOGIN_UNIFORM_RESPONSES": "true"}
	relay := map[string]string{"DEBUG": "true", "COOKIE_SECURE": "false"}
	clientIP := map[string]string{"HERALD_CLIENT_IP_HEADER": "X-Forwarded-For"}

	for _, tc := range []struct {
		name string
		env  map[string]string
		want []string
	}{
		{name: "mode on, client address forwarded to Herald", env: merge(on, warden, herald, clientIP)},
		{name: "mode on, client address not forwarded", env: merge(on, warden, herald), want: []string{sharedLimit}},
		{name: "mode on, blank client address header", env: merge(on, warden, herald, map[string]string{"HERALD_CLIENT_IP_HEADER": "  "}), want: []string{sharedLimit}},
		{name: "mode on with the debug-code relay", env: merge(on, warden, herald, clientIP, relay), want: []string{debugCode}},
		{name: "mode on with the debug-code relay and no client address", env: merge(on, warden, herald, relay), want: []string{debugCode, sharedLimit}},
		{name: "mode on with DEBUG over HTTPS", env: merge(on, warden, herald, clientIP, map[string]string{"DEBUG": "true", "COOKIE_SECURE": "true"})},
		{name: "mode on without Herald", env: merge(on, warden), want: []string{notOffered}},
		{name: "mode on without Warden", env: on, want: []string{notOffered}},
		{name: "mode on without Herald, relay settings", env: merge(on, warden, relay), want: []string{notOffered}},
		{name: "mode off with the debug-code relay and no client address", env: merge(warden, herald, relay)},
		{name: "mode off without Warden", env: map[string]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setUniformTestEnv(t, tc.env)
			var output bytes.Buffer
			log := logger.New(logger.Config{Level: logger.InfoLevel, Format: logger.FormatJSON, Output: &output})

			require.NoError(t, Initialize(log))
			for _, warning := range []string{notOffered, debugCode, sharedLimit} {
				expected := 0
				for _, wanted := range tc.want {
					if wanted == warning {
						expected = 1
					}
				}
				require.Equal(t, expected, bytes.Count(output.Bytes(), []byte(warning)), "%q in %s", warning, output.String())
			}
		})
	}
}
