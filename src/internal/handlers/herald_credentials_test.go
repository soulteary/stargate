package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestHeraldClientSelectsOneCredentialScheme(t *testing.T) {
	vars := []*config.EnvVariable{&config.HeraldEnabled, &config.HeraldURL, &config.HeraldAPIKey, &config.HeraldHMACSecret, &config.HeraldHMACKeyID, &config.HeraldTLSCACertFile, &config.HeraldTLSClientCert, &config.HeraldTLSClientKey, &config.HeraldTLSServerName}
	old := make([]string, len(vars))
	for i, v := range vars {
		old[i] = v.Value
	}
	t.Cleanup(func() {
		ResetHeraldClientForTest()
		for i, v := range vars {
			v.Value = old[i]
		}
	})
	for _, tc := range []struct {
		name, secret string
		wantError    bool
	}{{"dual credentials", "signing-secret", false}, {"invalid HMAC does not fall back", "wrong-secret", true}, {"API key only", "", false}} {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range vars {
				v.Value = ""
			}
			ResetHeraldClientForTest()
			config.HeraldEnabled.Value = "true"
			config.HeraldAPIKey.Value = "configured-api-key"
			config.HeraldHMACSecret.Value = tc.secret
			config.HeraldHMACKeyID.Value = "primary"
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Empty(t, r.Header.Get("Authorization"))
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				if tc.secret == "" {
					require.Equal(t, "configured-api-key", r.Header.Get("X-API-Key"))
					require.Empty(t, r.Header.Get("X-Signature"))
				} else {
					require.Empty(t, r.Header.Get("X-API-Key"))
					require.Equal(t, "v3", r.Header.Get("X-Signature-Version"))
					require.Equal(t, "primary", r.Header.Get("X-Key-Id"))
					require.NotEmpty(t, r.Header.Get("X-Nonce"))
					bodyHash := sha256.Sum256(body)
					idempotencyHash := sha256.Sum256([]byte(r.Header.Get("Idempotency-Key")))
					canonical := strings.Join([]string{"HERALD-HMAC-V3", r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("X-Timestamp"), r.Header.Get("X-Nonce"), r.Header.Get("X-Service"), r.Header.Get("X-Key-Id"), hex.EncodeToString(idempotencyHash[:]), hex.EncodeToString(bodyHash[:])}, "\n")
					mac := hmac.New(sha256.New, []byte("signing-secret"))
					_, _ = mac.Write([]byte(canonical))
					valid := hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(r.Header.Get("X-Signature")))
					if !valid {
						w.WriteHeader(401)
						_, _ = w.Write([]byte(`{"reason":"unauthorized"}`))
						return
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true,"challenge_id":"challenge","user_id":"user","purpose":"login","channel":"email"}`))
			}))
			defer server.Close()
			config.HeraldURL.Value = server.URL
			require.NoError(t, InitHeraldClient(testLogger()))
			_, err := getHeraldClient().VerifyChallengeV2(context.Background(), &herald.VerifyChallengeV2Request{ChallengeID: "challenge", Code: "123456", ExpectedUserID: "user", ExpectedPurpose: "login", ExpectedChannel: "email"})
			if tc.wantError {
				require.Error(t, err)
				var e *herald.HeraldError
				require.ErrorAs(t, err, &e)
				require.Equal(t, 401, e.StatusCode)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, int32(1), calls.Load())
		})
	}
}
