package handlers

import (
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/soulteary/stargate/src/internal/config"
)

func TestInitHeraldClientReturnsAndRemembersConfigurationError(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_URL", "https://herald.example.com")
	t.Setenv("HERALD_TLS_CA_CERT_FILE", filepath.Join(t.TempDir(), "missing-ca.pem"))

	ResetHeraldClientForTest()
	t.Cleanup(ResetHeraldClientForTest)
	testza.AssertNoError(t, config.Initialize(testLogger()))

	err := InitHeraldClient(testLogger())
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "create Herald client")
	testza.AssertNil(t, heraldClient)

	// sync.Once must not turn a failed first initialization into a later nil error.
	testza.AssertNotNil(t, InitHeraldClient(testLogger()))
}
