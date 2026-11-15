package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/soulteary/stargate/src/internal/config"
)

// WARDEN_HMAC_AUDIENCE selects the signature form of the requests Stargate
// sends to Warden: v3, which signs the audience, when it is set, and v2 when
// it is empty. The fake Warden recomputes the signature the way the Warden
// server does for the stated version.
func TestWardenRequestSignatureVersionFollowsHMACAudience(t *testing.T) {
	const keyID, secret = "stargate", "warden-hmac-secret-value"

	for _, tc := range []struct {
		name         string
		audience     string
		wantVersion  string
		wantAudience string
	}{
		{"no audience keeps v2", "", "v2", ""},
		{"audience selects v3", "warden-a", "v3", "warden-a"},
		{"audience is signed in normalized form", " Warden-A ", "v3", "warden-a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateGlobalState(t)
			requests := make(chan http.Header, 1)
			var path string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.EscapedPath()
				if r.URL.RawQuery != "" {
					path += "?" + r.URL.RawQuery
				}
				requests <- r.Header.Clone()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"user_id":"user-1","mail":"user@example.com","status":"active"}`))
			}))
			defer server.Close()

			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", "plaintext:test123")
			t.Setenv("WARDEN_ENABLED", "true")
			t.Setenv("WARDEN_URL", server.URL)
			t.Setenv("WARDEN_API_KEY", "")
			t.Setenv("WARDEN_HMAC_KEY_ID", keyID)
			t.Setenv("WARDEN_HMAC_SECRET", secret)
			t.Setenv("WARDEN_HMAC_AUDIENCE", tc.audience)
			ResetWardenClientForTesting()
			t.Cleanup(ResetWardenClientForTesting)
			testza.AssertNoError(t, config.Initialize(testLogger()))
			testza.AssertNoError(t, InitWardenClient(testLogger()))

			user, _, err := LookupUserInfo(context.Background(), "", "user@example.com")
			testza.AssertNoError(t, err)
			testza.AssertNotNil(t, user)

			header := <-requests
			testza.AssertEqual(t, tc.wantVersion, header.Get("X-Signature-Version"))
			testza.AssertEqual(t, keyID, header.Get("X-Key-Id"))

			emptyBody := sha256.Sum256(nil)
			canonical := strings.Join([]string{http.MethodGet, path, keyID, header.Get("X-Timestamp"), header.Get("X-Nonce"), hex.EncodeToString(emptyBody[:])}, "\n")
			if tc.wantVersion == "v3" {
				canonical = "v3\n" + tc.wantAudience + "\n" + canonical
			}
			mac := hmac.New(sha256.New, []byte(secret))
			_, _ = mac.Write([]byte(canonical))
			testza.AssertEqual(t, hex.EncodeToString(mac.Sum(nil)), header.Get("X-Signature"))
		})
	}
}
