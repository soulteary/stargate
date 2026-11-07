package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	challenge "github.com/soulteary/challenge-kit"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func TestBoundVerificationRecoversWithoutConsumingRetryableFailures(t *testing.T) {
	for _, failure := range []struct {
		name   string
		status int
	}{{"backend_unavailable", 503}, {"lock_contention", 409}} {
		t.Run(failure.name, func(t *testing.T) {
			oldClient, oldEnabled := heraldClient, config.HeraldEnabled.Value
			oldStore := getChallengeContextStore()
			t.Cleanup(func() {
				heraldClient = oldClient
				config.HeraldEnabled.Value = oldEnabled
				SetChallengeContextStore(oldStore)
			})
			config.HeraldEnabled.Value = "true"
			mr := miniredis.RunT(t)
			good := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			t.Cleanup(func() { _ = good.Close() })
			bad := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			require.NoError(t, bad.Close())
			cfg := challenge.DefaultConfig()
			cfg.RequireActiveVerification = true
			cfg.VerifyLockWait = 5 * time.Millisecond
			cfg.VerifyLockRetry = time.Millisecond
			manager, unavailable := challenge.NewManager(good, cfg), challenge.NewManager(bad, cfg)
			ch, code, err := manager.Create(context.Background(), challenge.CreateRequest{UserID: "user-1", Channel: challenge.ChannelEmail, Destination: "alice@example.com", Purpose: "login"})
			require.NoError(t, err)
			_, err = manager.SwapActive(context.Background(), ch)
			require.NoError(t, err)
			seedLoginChallengeBinding(t, ch.ID, ch.UserID, "email", ch.Destination)
			var outage atomic.Bool
			outage.Store(true)
			if failure.name == "lock_contention" {
				require.NoError(t, good.Set(context.Background(), cfg.VerifyLockPrefix+ch.ID, "held-by-another-request", time.Minute).Err())
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v2/otp/verifications", r.URL.Path)
				var request herald.VerifyChallengeV2Request
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				current := manager
				if outage.Load() && failure.name == "backend_unavailable" {
					current = unavailable
				}
				result, verifyErr := current.VerifyWithOptions(r.Context(), request.ChallengeID, request.Code, "", challenge.VerifyOptions{ExpectedUserID: request.ExpectedUserID, ExpectedPurpose: request.ExpectedPurpose, ExpectedChannel: challenge.Channel(request.ExpectedChannel)})
				w.Header().Set("Content-Type", "application/json")
				if verifyErr != nil {
					require.Equal(t, failure.name, result.Reason)
					w.Header().Set("Retry-After", "1")
					w.WriteHeader(failure.status)
					_ = json.NewEncoder(w).Encode(herald.VerifyChallengeV2Response{Reason: result.Reason})
					return
				}
				_ = json.NewEncoder(w).Encode(herald.VerifyChallengeV2Response{OK: true, ChallengeID: ch.ID, UserID: ch.UserID, Purpose: ch.Purpose, Channel: "email"})
			}))
			defer server.Close()
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)
			binding, _, err := getChallengeContextStore().Get(context.Background(), ch.ID)
			require.NoError(t, err)
			identity := &loginIdentity{userID: ch.UserID, wardenUserInfo: &warden.AllowListUser{UserID: ch.UserID, Mail: ch.Destination, Status: "active"}}
			ctx, app := createTestContext("POST", "/_login", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "challenge_id="+ch.ID+"&verify_code="+code)
			ok, err := identity.verifyChallenge(ctx, ctx.Context(), binding, false)
			require.NoError(t, err)
			require.False(t, ok)
			require.Equal(t, failure.status, ctx.Response().StatusCode())
			require.Empty(t, string(ctx.Response().Header.Peek("Retry-After")))
			app.ReleaseCtx(ctx)
			_, err = manager.Get(context.Background(), ch.ID)
			require.NoError(t, err)
			_, found, err := getChallengeContextStore().Get(context.Background(), ch.ID)
			require.NoError(t, err)
			require.True(t, found)
			outage.Store(false)
			require.NoError(t, good.Del(context.Background(), cfg.VerifyLockPrefix+ch.ID).Err())
			ctx, app = createTestContext("POST", "/_login", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "challenge_id="+ch.ID+"&verify_code="+code)
			ok, err = identity.verifyChallenge(ctx, ctx.Context(), binding, false)
			require.NoError(t, err)
			require.True(t, ok)
			app.ReleaseCtx(ctx)
			_, err = manager.Get(context.Background(), ch.ID)
			require.Error(t, err)
			_, found, err = getChallengeContextStore().Get(context.Background(), ch.ID)
			require.NoError(t, err)
			require.False(t, found)
		})
	}
}

func TestVerificationErrorClassificationPreservesProofRejection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		response *herald.VerifyChallengeResponse
		want     int
	}{
		{name: "wrapped upstream failure", err: fmt.Errorf("provider: %w", &herald.HeraldError{StatusCode: 502, Reason: "backend_unavailable"}), want: 503},
		{name: "binding mismatch", err: &herald.HeraldError{StatusCode: 409, Reason: "context_mismatch"}, want: 401},
		{name: "unknown conflict", err: &herald.HeraldError{StatusCode: 409, Reason: "other"}, want: 401},
		{name: "service credential failure", err: &herald.HeraldError{StatusCode: 401, Reason: "unauthorized"}, want: 401},
		{name: "invalid proof", err: &herald.HeraldError{StatusCode: 400, Reason: "invalid"}, response: &herald.VerifyChallengeResponse{Reason: "invalid"}, want: 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, app := createTestContext("POST", "/_login", nil, "")
			defer app.ReleaseCtx(ctx)
			require.NoError(t, handleChallengeVerificationError(ctx, tc.err, tc.response, challengeContext{}, false))
			require.Equal(t, tc.want, ctx.Response().StatusCode())
			require.Empty(t, string(ctx.Response().Header.Peek("Retry-After")), "the current SDK does not expose provider retry headers")
		})
	}
}
