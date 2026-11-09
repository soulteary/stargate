package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func TestStrictDingtalkDestination(t *testing.T) {
	old := config.LoginDingtalkRequireUserID.Value
	t.Cleanup(func() { config.LoginDingtalkRequireUserID.Value = old })
	for _, tc := range []struct{ name, strict, id, phone, want string }{
		{"legacy phone", "false", "", "13800138000", "13800138000"},
		{"legacy userid", "false", " Ding-A_Z9 ", "13800138000", "Ding-A_Z9"},
		{"strict mapped case", "true", " Ding-A_Z9 ", "13800138000", "Ding-A_Z9"},
		{"strict missing", "true", "", "13800138000", ""},
		{"strict whitespace", "true", " \t ", "13800138000", ""},
		{"strict comma", "true", "Ding-A,Ding-B", "13800138000", ""},
		{"strict CR", "true", "Ding-A\r", "13800138000", ""},
		{"strict LF", "true", "Ding-A\nDing-B", "13800138000", ""},
		{"strict internal control", "true", "Ding-A\x00Ding-B", "13800138000", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config.LoginDingtalkRequireUserID.Value = tc.strict
			channel, dest, reason := selectVerificationDestination(&warden.AllowListUser{DingtalkUserID: tc.id, Phone: tc.phone, Mail: "synthetic@example.com"}, "dingtalk")
			require.Equal(t, tc.want, dest)
			if tc.want == "" {
				require.Equal(t, "dingtalk_not_bound", reason)
				require.Empty(t, channel)
			} else {
				require.Equal(t, "dingtalk", channel)
				require.Empty(t, reason)
			}
		})
	}
	config.LoginDingtalkRequireUserID.Value = "true"
	_, _, reason := selectVerificationDestination(nil, "dingtalk")
	require.Equal(t, "destination_unavailable", reason)
}

func TestStrictDingtalkPriorBindingRejectsPhoneOrChangedMapping(t *testing.T) {
	old := config.LoginDingtalkRequireUserID.Value
	t.Cleanup(func() { config.LoginDingtalkRequireUserID.Value = old })
	identity := loginIdentity{userID: "synthetic-user", wardenUserInfo: &warden.AllowListUser{UserID: "synthetic-user", Phone: "13800138000", DingtalkUserID: "Ding-A_Z9", Status: "active"}}
	binding := challengeContext{ChallengeID: "synthetic-challenge", UserID: "synthetic-user", Channel: "dingtalk", Destination: "Ding-A_Z9"}
	config.LoginDingtalkRequireUserID.Value = "true"
	require.True(t, identity.matchesLoginChallenge(binding.ChallengeID, binding))
	for _, destination := range []string{"13800138000", "ding-a_z9", "Ding-B_Z9"} {
		binding.Destination = destination
		require.False(t, identity.matchesLoginChallenge(binding.ChallengeID, binding))
	}
	identity.wardenUserInfo.DingtalkUserID = ""
	binding.Destination = "13800138000"
	require.False(t, identity.matchesLoginChallenge(binding.ChallengeID, binding))
	config.LoginDingtalkRequireUserID.Value = "false"
	require.True(t, identity.matchesLoginChallenge(binding.ChallengeID, binding))
}

func TestStrictDingtalkCreationHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, status, mapping, channel string
		missing, unavailable           bool
		want                           int
	}{
		{name: "mapped ignores supplied receiver", status: "active", mapping: " Ding-A_Z9 ", channel: "dingtalk", want: 200},
		{name: "unmapped phone cannot fallback", status: "active", channel: "dingtalk", want: 400},
		{name: "whitespace cannot fallback", status: "active", mapping: " \t ", channel: "dingtalk", want: 400},
		{name: "comma cannot fanout", status: "active", mapping: "Ding-A,Ding-B", channel: "dingtalk", want: 400},
		{name: "linebreak cannot fanout", status: "active", mapping: "Ding-A\nDing-B", channel: "dingtalk", want: 400},
		{name: "inactive rejects", status: "inactive", mapping: "Ding-A_Z9", channel: "dingtalk", want: 401},
		{name: "suspended rejects", status: "suspended", mapping: "Ding-A_Z9", channel: "dingtalk", want: 401},
		{name: "missing rejects", missing: true, channel: "dingtalk", want: 401},
		{name: "source unavailable keeps native refusal", unavailable: true, channel: "dingtalk", want: 401},
		{name: "SMS profile off", status: "active", mapping: "Ding-A_Z9", channel: "sms", want: 400},
		{name: "email profile off", status: "active", mapping: "Ding-A_Z9", channel: "email", want: 400},
		{name: "blank profile no fallback", status: "active", mapping: "Ding-A_Z9", want: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int64
			directory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tc.unavailable {
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(`{"error":"synthetic_directory_unavailable"}`))
					return
				}
				if tc.missing {
					w.WriteHeader(404)
					_, _ = w.Write([]byte(`{"error":"not_found"}`))
					return
				}
				_ = json.NewEncoder(w).Encode(warden.AllowListUser{UserID: "synthetic-user", Phone: "13800138000", Mail: "synthetic@example.com", Status: tc.status, DingtalkUserID: tc.mapping})
			}))
			defer directory.Close()
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				var req herald.CreateChallengeRequest
				if r.URL.Path != "/v1/otp/challenges" || json.Unmarshal(raw, &req) != nil || req.UserID != "synthetic-user" || req.Channel != "dingtalk" || req.Destination != "Ding-A_Z9" {
					t.Error("unexpected canonical challenge or verification side effect")
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(herald.CreateChallengeResponse{ChallengeID: "synthetic-challenge", ExpiresIn: 120, NextResendIn: 30})
			}))
			defer provider.Close()
			setupSendVerifyCodeBaseEnv(t)
			for key, value := range map[string]string{"WARDEN_URL": directory.URL, "WARDEN_HMAC_SECRET": "", "WARDEN_HMAC_KEY_ID": "", "WARDEN_SOURCE_MAX_AGE": "0s", "HERALD_ENABLED": "true", "HERALD_URL": provider.URL, "HERALD_API_KEY": "synthetic-only", "HERALD_HMAC_SECRET": "", "LOGIN_DINGTALK_REQUIRE_USERID": "true", "LOGIN_SMS_ENABLED": "false", "LOGIN_EMAIL_ENABLED": "false"} {
				t.Setenv(key, value)
			}
			auth.ResetWardenClientForTesting()
			resetHeraldClientForTesting()
			previous := getChallengeContextStore()
			SetChallengeContextStore(newMemoryChallengeContextStore())
			t.Cleanup(func() {
				auth.ResetWardenClientForTesting()
				resetHeraldClientForTesting()
				SetChallengeContextStore(previous)
				config.LoginDingtalkRequireUserID.Value = "false"
			})
			log := testLoggerSendVerifyCode()
			require.NoError(t, config.Initialize(log))
			require.NoError(t, auth.InitWardenClient(log))
			require.NoError(t, InitHeraldClient(log))
			app := fiber.New()
			app.Post("/_send_verify_code", SendVerifyCodeAPI())
			form := url.Values{"phone": {"13800138000"}, "deliver_via": {tc.channel}, "destination": {"attacker"}, "dingtalk_userid": {"attacker"}, "user_id": {"attacker"}}
			// Repeated identical missing/invalid mappings must remain side-effect free.
			for attempt := 0; attempt < 2; attempt++ {
				if tc.want == 200 && attempt > 0 {
					break
				}
				req := httptest.NewRequest("POST", "/_send_verify_code", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Idempotency-Key", "synthetic-retry-key")
				resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
				require.NoError(t, err)
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				require.Equal(t, tc.want, resp.StatusCode, string(body))
			}
			// A pre-opt-in phone-derived login binding must be rejected before
			// calling or consuming any Herald verification, including when the
			// current directory mapping exists but no longer equals that contact.
			oldBinding := challengeContext{ChallengeID: "previous-phone-challenge", UserID: "synthetic-user", Channel: "dingtalk", Destination: "13800138000"}
			stored, err := getChallengeContextStore().PutIfAbsent(context.Background(), oldBinding, time.Minute)
			require.NoError(t, err)
			require.True(t, stored)
			app.Post("/_login", LoginAPI(setupTestStore()))
			loginForm := url.Values{"auth_method": {"warden"}, "phone": {"13800138000"}, "challenge_id": {oldBinding.ChallengeID}, "verify_code": {"123456"}}
			loginRequest := httptest.NewRequest("POST", "/_login", strings.NewReader(loginForm.Encode()))
			loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			loginRequest.Header.Set("Accept", "application/json")
			loginResponse, err := app.Test(loginRequest, fiber.TestConfig{Timeout: 5 * time.Second})
			require.NoError(t, err)
			loginBody, err := io.ReadAll(loginResponse.Body)
			require.NoError(t, err)
			require.NoError(t, loginResponse.Body.Close())
			loginStatus := http.StatusUnauthorized
			if tc.unavailable {
				loginStatus = http.StatusServiceUnavailable
			}
			require.Equal(t, loginStatus, loginResponse.StatusCode, string(loginBody))
			remaining, found, err := getChallengeContextStore().Get(context.Background(), oldBinding.ChallengeID)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, oldBinding, remaining)
			if tc.want == 200 {
				require.Equal(t, int64(1), calls.Load())
			} else {
				require.Zero(t, calls.Load())
			}
		})
	}
}
