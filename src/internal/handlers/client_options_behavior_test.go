package handlers

import (
	"testing"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestHeraldInitializationValidatesCredentialsAndTLSOptions(t *testing.T) {
	variables := []*config.EnvVariable{&config.HeraldEnabled, &config.HeraldURL, &config.HeraldAPIKey, &config.HeraldHMACSecret, &config.HeraldHMACKeyID, &config.HeraldTLSCACertFile, &config.HeraldTLSClientCert, &config.HeraldTLSClientKey, &config.HeraldTLSServerName}
	old := make([]string, len(variables))
	for i, v := range variables {
		old[i] = v.Value
	}
	t.Cleanup(func() {
		ResetHeraldClientForTest()
		for i, v := range variables {
			v.Value = old[i]
		}
	})
	for _, tc := range []struct {
		name      string
		values    map[*config.EnvVariable]string
		wantError bool
	}{
		{"disabled", nil, false},
		{"missing URL", map[*config.EnvVariable]string{&config.HeraldEnabled: "true"}, true},
		{"HMAC", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldHMACSecret: "test-hmac-secret", &config.HeraldHMACKeyID: "primary"}, false},
		{"certificate without key", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldAPIKey: "test-api-key", &config.HeraldTLSClientCert: "missing.pem"}, false},
		{"TLS options", map[*config.EnvVariable]string{&config.HeraldEnabled: "true", &config.HeraldURL: "https://herald.example.com", &config.HeraldTLSCACertFile: "missing-ca.pem", &config.HeraldTLSClientCert: "missing.pem", &config.HeraldTLSClientKey: "missing.key", &config.HeraldTLSServerName: "herald.example.com"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ResetHeraldClientForTest()
			for _, v := range variables {
				v.Value = ""
			}
			for v, value := range tc.values {
				v.Value = value
			}
			err := InitHeraldClient(testLogger())
			if tc.wantError {
				require.Error(t, err)
				require.Nil(t, getHeraldClient())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
