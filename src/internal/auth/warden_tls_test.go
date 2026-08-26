package auth

import (
	"crypto/tls"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestBuildWardenTLSConfigDisabled(t *testing.T) {
	cfg, err := buildWardenTLSConfig("", "", "", "")
	testza.AssertNoError(t, err)
	testza.AssertNil(t, cfg)
}

func TestBuildWardenTLSConfigRequiresCertificatePair(t *testing.T) {
	_, err := buildWardenTLSConfig("", "client.crt", "", "")
	testza.AssertNotNil(t, err)
}

func TestBuildWardenTLSConfigSetsSecureDefaults(t *testing.T) {
	cfg, err := buildWardenTLSConfig("", "", "", "warden.internal")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, uint16(tls.VersionTLS12), cfg.MinVersion)
	testza.AssertEqual(t, "warden.internal", cfg.ServerName)
}
