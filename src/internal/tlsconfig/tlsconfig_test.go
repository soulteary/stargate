package tlsconfig

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
)

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
