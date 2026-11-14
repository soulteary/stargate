package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/soulteary/stargate/src/internal/config"
)

// WARDEN_CACHE_TTL is documented as having no effect on per-identifier
// lookups: authorization decisions are never served from a cache, so a user
// disabled in Warden is seen by the next lookup. This pins that behaviour.
func TestLookupUserInfoIsNotCachedByWardenCacheTTL(t *testing.T) {
	var requests atomic.Int64
	var status atomic.Value
	status.Store("active")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"user_id": "user-1", "mail": "user@example.com", "status": status.Load().(string),
		})
	}))
	defer server.Close()

	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	t.Setenv("WARDEN_URL", server.URL)
	t.Setenv("WARDEN_CACHE_TTL", "3600")
	ResetWardenClientForTesting()
	t.Cleanup(ResetWardenClientForTesting)
	testza.AssertNoError(t, config.Initialize(testLogger()))
	testza.AssertNoError(t, InitWardenClient(testLogger()))

	for i := 1; i <= 3; i++ {
		user, _, err := LookupUserInfo(context.Background(), "", "user@example.com")
		testza.AssertNoError(t, err)
		testza.AssertNotNil(t, user)
		testza.AssertEqual(t, int64(i), requests.Load())
	}

	status.Store("inactive")
	user, _, err := LookupUserInfo(context.Background(), "", "user@example.com")
	testza.AssertNil(t, user)
	testza.AssertTrue(t, err == ErrUserUnavailable)
	testza.AssertEqual(t, int64(4), requests.Load())
}
