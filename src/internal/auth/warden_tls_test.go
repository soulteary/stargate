package auth

import (
	"crypto/tls"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestBuildWardenTLSConfigRejectsUnreadableAndInvalidCA(t *testing.T) {
	_, err := buildWardenTLSConfig(filepath.Join(t.TempDir(), "missing.pem"), "", "", "")
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "read Warden CA certificate")

	invalidCA := filepath.Join(t.TempDir(), "invalid.pem")
	testza.AssertNoError(t, os.WriteFile(invalidCA, []byte("not a certificate"), 0o600))
	_, err = buildWardenTLSConfig(invalidCA, "", "", "")
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "contains no valid certificates")
}

func TestBuildWardenTLSConfigLoadsCA(t *testing.T) {
	server := httptest.NewTLSServer(nil)
	defer server.Close()

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.TLS.Certificates[0].Certificate[0]})
	testza.AssertNoError(t, os.WriteFile(caFile, certificate, 0o600))

	cfg, err := buildWardenTLSConfig(caFile, "", "", "warden.internal")
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, cfg.RootCAs)
	testza.AssertEqual(t, "warden.internal", cfg.ServerName)
}

func TestBuildWardenTLSConfigRejectsInvalidClientPair(t *testing.T) {
	tmp := t.TempDir()
	certFile := filepath.Join(tmp, "client.pem")
	keyFile := filepath.Join(tmp, "client.key")
	testza.AssertNoError(t, os.WriteFile(certFile, []byte("invalid"), 0o600))
	testza.AssertNoError(t, os.WriteFile(keyFile, []byte("invalid"), 0o600))

	_, err := buildWardenTLSConfig("", certFile, keyFile, "")
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "load Warden client certificate")
}
