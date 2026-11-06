package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	challenge "github.com/soulteary/challenge-kit"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func seedLoginChallengeBinding(t *testing.T, id, user, channel, destination string) {
	t.Helper()
	previous := getChallengeContextStore()
	store := newMemoryChallengeContextStore()
	SetChallengeContextStore(store)
	t.Cleanup(func() { _ = store.Delete(context.Background(), id); SetChallengeContextStore(previous) })
	_, err := store.PutIfAbsent(context.Background(), challengeContext{ChallengeID: id, UserID: user, Channel: channel, Destination: destination}, time.Minute)
	require.NoError(t, err)
}

// The local HTTP adapter uses the released challenge manager's bound verifier.
// This checks the gateway's actual SDK request against the provider contract,
// including that a context mismatch does not consume a correct code.
func TestLoginChallengeUsesBoundProviderContract(t *testing.T) {
	for _, tc := range []struct {
		name, purpose, corrupt           string
		missing, staleContact, redisPeer bool
		status, calls                    int
	}{
		{name: "email login", purpose: "login", status: 200, calls: 1},
		{name: "shared Redis peer", purpose: "login", redisPeer: true, status: 200, calls: 1},
		{name: "non-login purpose", purpose: "profile_confirmation", status: 401, calls: 1},
		{name: "missing binding", purpose: "login", missing: true, status: 401},
		{name: "changed directory contact", purpose: "login", staleContact: true, status: 401},
		{name: "response ID", purpose: "login", corrupt: "id", status: 401, calls: 1},
		{name: "response subject", purpose: "login", corrupt: "user", status: 401, calls: 1},
		{name: "response purpose", purpose: "login", corrupt: "purpose", status: 401, calls: 1},
		{name: "response channel", purpose: "login", corrupt: "channel", status: 401, calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldClient, oldEnabled, oldEmail, oldStore := heraldClient, config.HeraldEnabled.Value, config.LoginEmailEnabled.Value, getChallengeContextStore()
			t.Cleanup(func() {
				heraldClient = oldClient
				config.HeraldEnabled.Value = oldEnabled
				config.LoginEmailEnabled.Value = oldEmail
				SetChallengeContextStore(oldStore)
			})
			config.HeraldEnabled.Value, config.LoginEmailEnabled.Value = "true", "true"
			mr := miniredis.RunT(t)
			rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
			t.Cleanup(func() { _ = rc.Close() })
			cfg := challenge.DefaultConfig()
			cfg.RequireActiveVerification = true
			manager := challenge.NewManager(rc, cfg)
			ch, code, err := manager.Create(context.Background(), challenge.CreateRequest{UserID: "user-1", Channel: challenge.ChannelEmail, Destination: "alice@example.com", Purpose: tc.purpose})
			require.NoError(t, err)
			_, err = manager.SwapActive(context.Background(), ch)
			require.NoError(t, err)
			binding := challengeContext{ChallengeID: ch.ID, UserID: ch.UserID, Channel: "email", Destination: ch.Destination}
			var writer ChallengeContextStore = newMemoryChallengeContextStore()
			reader := writer
			if tc.redisPeer {
				peer := redis.NewClient(&redis.Options{Addr: mr.Addr()})
				t.Cleanup(func() { _ = peer.Close() })
				writer, reader = NewChallengeContextStore(rc), NewChallengeContextStore(peer)
			}
			if !tc.missing {
				_, err = writer.PutIfAbsent(context.Background(), binding, time.Minute)
				require.NoError(t, err)
			}
			SetChallengeContextStore(reader)
			t.Cleanup(func() { _ = writer.Delete(context.Background(), ch.ID) })
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "/v2/otp/verifications", r.URL.Path)
				var request herald.VerifyChallengeV2Request
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, "user-1", request.ExpectedUserID)
				require.Equal(t, "login", request.ExpectedPurpose)
				require.Equal(t, "email", request.ExpectedChannel)
				result, verifyErr := manager.VerifyWithOptions(context.Background(), request.ChallengeID, request.Code, request.ClientIP, challenge.VerifyOptions{ExpectedUserID: request.ExpectedUserID, ExpectedPurpose: request.ExpectedPurpose, ExpectedChannel: challenge.Channel(request.ExpectedChannel)})
				w.Header().Set("Content-Type", "application/json")
				if verifyErr != nil || !result.OK {
					w.WriteHeader(400)
					_ = json.NewEncoder(w).Encode(herald.VerifyChallengeV2Response{Reason: string(result.Reason)})
					return
				}
				response := herald.VerifyChallengeV2Response{OK: true, UserID: result.Challenge.UserID, ChallengeID: result.Challenge.ID, Purpose: result.Challenge.Purpose, Channel: string(result.Challenge.Channel), AMR: []string{"otp", "email"}}
				switch tc.corrupt {
				case "id":
					response.ChallengeID = "other"
				case "user":
					response.UserID = "other"
				case "purpose":
					response.Purpose = "other"
				case "channel":
					response.Channel = "sms"
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)
			app := fiber.New()
			app.Post("/", func(c fiber.Ctx) error {
				current, loadErr := loadChallengeContext(c.Context(), ch.ID)
				require.NoError(t, loadErr)
				mail := "alice@example.com"
				if tc.staleContact {
					mail = "changed@example.com"
				}
				identity := &loginIdentity{userID: "user-1", wardenUserInfo: &warden.AllowListUser{UserID: "user-1", Mail: mail}}
				ok, verifyErr := identity.verifyChallenge(c, c.Context(), current, false)
				if !ok {
					return verifyErr
				}
				return c.SendStatus(200)
			})
			request := httptest.NewRequest("POST", "/", strings.NewReader("challenge_id="+ch.ID+"&verify_code="+code))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err := app.Test(request)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, tc.status, response.StatusCode)
			require.Equal(t, int32(tc.calls), calls.Load())
			if tc.calls == 0 || tc.purpose != "login" {
				remaining, getErr := manager.Get(context.Background(), ch.ID)
				require.NoError(t, getErr)
				require.NotNil(t, remaining)
			}
		})
	}
}
