package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
)

func TestBuildDefaultsAndCertificateFailures(t *testing.T) {
	cfg, err := Build("test", "", "", "", "")
	if err != nil || cfg != nil {
		t.Fatalf("default TLS: config=%v err=%v", cfg, err)
	}
	invalidCA := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(invalidCA, []byte("invalid certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][4]string{
		{invalidCA + ".missing", "", "", ""},
		{invalidCA, "", "", ""},
		{"", "missing.pem", "missing.key", ""},
		{"", "", "missing.key", ""},
	} {
		if _, err := Build("test", args[0], args[1], args[2], args[3]); err == nil {
			t.Fatalf("accepted invalid certificate configuration: %v", args)
		}
		if _, err := HTTPClient("test", args[0], args[1], args[2], args[3], time.Second); err == nil {
			t.Fatalf("HTTP client accepted invalid certificate configuration: %v", args)
		}
	}
}

func TestBuildLoadsClientCertificate(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	cert := server.TLS.Certificates[0]
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	certFile := filepath.Join(t.TempDir(), "client.pem")
	keyFile := filepath.Join(t.TempDir(), "client.key")
	for path, data := range map[string][]byte{
		certFile: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}),
		keyFile:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}),
	} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Build("test", "", certFile, keyFile, "service.example")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinVersion != tls.VersionTLS12 || cfg.ServerName != "service.example" || len(cfg.Certificates) != 1 {
		t.Fatalf("client TLS configuration was not retained: %+v", cfg)
	}
}

func TestHTTPClientUsesConfiguredCA(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.TLS.Certificates[0].Certificate[0]})
	testza.AssertNoError(t, os.WriteFile(caFile, certificate, 0o600))

	client, err := HTTPClient("test service", caFile, "", "", "", time.Second)
	testza.AssertNoError(t, err)
	resp, err := client.Get(server.URL)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, http.StatusOK, resp.StatusCode)
	testza.AssertNoError(t, resp.Body.Close())
}

func TestBuildRejectsIncompleteClientCertificatePair(t *testing.T) {
	_, err := Build("test service", "", "client.pem", "", "")
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "configured together")
}
