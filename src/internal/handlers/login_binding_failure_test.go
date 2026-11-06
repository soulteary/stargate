package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoginDestinationComparisonPreservesContactIdentity(t *testing.T) {
	for _, tc := range []struct {
		channel, left, right string
		want                 bool
	}{
		{"email", "Alice@EXAMPLE.COM", "Alice@example.com", true},
		{"email", "Alice@example.com", "alice@example.com", false},
		{"email", "alice+tag@example.com", "alice@example.com", false},
		{"email", "broken-address", "different-address", false},
		{"sms", "138 0013-8000", "13800138000", true},
		{"sms", "+86 13800138000", "13800138000", false},
		{"sms", "13800138000", "13900139000", false},
		{"sms", "invalid", "different", false},
		{"dingtalk", "account-1", "account-1", true},
		{"dingtalk", "account-1", "ACCOUNT-1", false},
		{"email", "", "alice@example.com", false},
	} {
		t.Run(tc.channel+tc.left+tc.right, func(t *testing.T) {
			require.Equal(t, tc.want, sameVerificationDestination(tc.channel, tc.left, tc.right))
		})
	}
}

type failedBindingStore struct {
	ChallengeContextStore
	stored, found  bool
	putErr, getErr error
	existing       challengeContext
}

func (s failedBindingStore) PutIfAbsent(context.Context, challengeContext, time.Duration) (bool, error) {
	return s.stored, s.putErr
}
func (s failedBindingStore) Get(context.Context, string) (challengeContext, bool, error) {
	return s.existing, s.found, s.getErr
}

func TestSendingRequiresDurableMatchingLoginBinding(t *testing.T) {
	for _, scenario := range []string{"write failure", "read failure", "missing replay", "other challenge", "other subject", "other channel", "changed destination", "matching replay", "new record"} {
		t.Run(scenario, func(t *testing.T) {
			variables := []*config.EnvVariable{&config.WardenEnabled, &config.WardenURL, &config.WardenAPIKey, &config.WardenHMACSecret, &config.WardenHMACKeyID, &config.WardenSourceMaxAge, &config.HeraldEnabled, &config.LoginEmailEnabled}
			old := make([]string, len(variables))
			for i, v := range variables {
				old[i] = v.Value
			}
			previousClient, previousStore := heraldClient, getChallengeContextStore()
			t.Cleanup(func() {
				heraldClient = previousClient
				SetChallengeContextStore(previousStore)
				for i, v := range variables {
					v.Value = old[i]
				}
				auth.ResetWardenClientForTesting()
			})
			directory := newWardenUserServer(t, "", "alice@example.com")
			defer directory.Close()
			config.WardenEnabled.Value, config.WardenURL.Value = "true", directory.URL
			config.WardenAPIKey.Value, config.WardenHMACSecret.Value, config.WardenHMACKeyID.Value, config.WardenSourceMaxAge.Value = "", "", "", ""
			config.HeraldEnabled.Value, config.LoginEmailEnabled.Value = "true", "true"
			auth.ResetWardenClientForTesting()
			require.NoError(t, auth.InitWardenClient(testLogger()))
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v1/otp/challenges", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(herald.CreateChallengeResponse{ChallengeID: "binding-challenge", ExpiresIn: 120, NextResendIn: 30})
			}))
			defer provider.Close()
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(provider.URL))
			require.NoError(t, err)
			binding := failedBindingStore{found: true, existing: challengeContext{ChallengeID: "binding-challenge", UserID: "user-1", Channel: "email", Destination: "alice@EXAMPLE.COM"}}
			want := 503
			switch scenario {
			case "write failure":
				binding.putErr = errors.New("injected write failure")
			case "read failure":
				binding.getErr = errors.New("injected read failure")
			case "missing replay":
				binding.found = false
			case "other challenge":
				binding.existing.ChallengeID = "other"
			case "other subject":
				binding.existing.UserID = "other"
			case "other channel":
				binding.existing.Channel = "sms"
			case "changed destination":
				binding.existing.Destination = "changed@example.com"
			case "matching replay":
				want = 200
			case "new record":
				binding.stored = true
				want = 200
			}
			SetChallengeContextStore(binding)
			ctx, app := createTestContext("POST", "/_send_verify_code", map[string]string{"Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json"}, "mail=alice@example.com&deliver_via=email")
			defer app.ReleaseCtx(ctx)
			require.NoError(t, SendVerifyCodeAPI()(ctx))
			require.Equal(t, want, ctx.Response().StatusCode(), string(ctx.Response().Body()))
			if want == 503 {
				require.Contains(t, string(ctx.Response().Body()), "binding_unavailable")
				require.NotContains(t, string(ctx.Response().Body()), "binding-challenge")
			} else {
				require.True(t, strings.Contains(string(ctx.Response().Body()), "binding-challenge"))
			}
		})
	}
}

func TestLoginAPIRejectsUnavailableChallengeContextBeforeProviderCall(t *testing.T) {
	for _, scenario := range []string{"missing", "backend failure"} {
		t.Run(scenario, func(t *testing.T) {
			variables := []*config.EnvVariable{&config.WardenEnabled, &config.WardenURL, &config.WardenAPIKey, &config.WardenHMACSecret, &config.WardenHMACKeyID, &config.WardenSourceMaxAge, &config.HeraldEnabled, &config.HeraldTOTPEnabled}
			old := make([]string, len(variables))
			for i, v := range variables {
				old[i] = v.Value
			}
			previousClient, previousStore := heraldClient, getChallengeContextStore()
			t.Cleanup(func() {
				heraldClient = previousClient
				SetChallengeContextStore(previousStore)
				for i, v := range variables {
					v.Value = old[i]
				}
				auth.ResetWardenClientForTesting()
			})
			directory := newWardenUserServer(t, "", "alice@example.com")
			defer directory.Close()
			config.WardenEnabled.Value, config.WardenURL.Value = "true", directory.URL
			config.WardenAPIKey.Value, config.WardenHMACSecret.Value, config.WardenHMACKeyID.Value, config.WardenSourceMaxAge.Value = "", "", "", ""
			config.HeraldEnabled.Value, config.HeraldTOTPEnabled.Value = "true", "false"
			auth.ResetWardenClientForTesting()
			require.NoError(t, auth.InitWardenClient(testLogger()))
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("unbound login reached provider")
				w.WriteHeader(500)
			}))
			defer provider.Close()
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(provider.URL))
			require.NoError(t, err)
			binding := failedBindingStore{}
			want := 401
			if scenario == "backend failure" {
				binding.getErr = errors.New("injected context backend failure")
				want = 503
			}
			SetChallengeContextStore(binding)
			ctx, app := createTestContext("POST", "/_login", map[string]string{"Host": "auth.example.com", "Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json"}, "auth_method=warden&mail=alice@example.com&challenge_id=missing-challenge&verify_code=123456")
			defer app.ReleaseCtx(ctx)
			require.NoError(t, LoginAPI(setupTestStore())(ctx))
			require.Equal(t, want, ctx.Response().StatusCode(), string(ctx.Response().Body()))
			require.NotContains(t, string(ctx.Response().Body()), `"success":true`)
		})
	}
}
