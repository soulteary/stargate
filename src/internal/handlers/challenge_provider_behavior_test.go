package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

type unavailableContextDelete struct{ ChallengeContextStore }

func (unavailableContextDelete) Delete(context.Context, string) error {
	return errors.New("injected context delete failure")
}

func TestChallengeProviderFailureAndCommittedSuccess(t *testing.T) {
	previousClient, previousEnabled := heraldClient, config.HeraldEnabled.Value
	t.Cleanup(func() {
		heraldClient = previousClient
		config.HeraldEnabled.Value = previousEnabled
		SetChallengeContextStore(nil)
	})
	config.HeraldEnabled.Value = "true"
	for _, tc := range []struct {
		name, reason string
		status       int
		ok           bool
	}{
		{"invalid code", "invalid", 200, false},
		{"missing reason", "", 200, false},
		{"provider rejects code", "invalid", 400, false},
		{"consumed with failed audit cleanup", "", 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(herald.VerifyChallengeResponse{OK: tc.ok, Reason: tc.reason, UserID: "user-1"})
			}))
			defer server.Close()
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)
			SetChallengeContextStore(unavailableContextDelete{})
			ctx, app := createTestContext("POST", "/_login", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "challenge_id=challenge-1&verify_code=123456")
			defer app.ReleaseCtx(ctx)
			identity := &loginIdentity{userID: "user-1"}
			ok, err := identity.verifyChallenge(ctx, context.Background(), challengeContext{ChallengeID: "challenge-1", UserID: "user-1"}, false)
			require.NoError(t, err)
			require.Equal(t, tc.ok, ok)
			if !tc.ok {
				require.Equal(t, 401, ctx.Response().StatusCode())
			}
		})
	}
}
