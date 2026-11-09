package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

func TestFormalThreeServiceBridge(t *testing.T) {
	endpoints := map[string]string{}
	for _, name := range []string{"FORMAL_HERALD_URL", "FORMAL_ADAPTER_URL"} {
		s := os.Getenv(name)
		u, e := url.Parse(s)
		if e != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.Path != "" || u.RawQuery != "" || u.User != nil || u.Fragment != "" {
			t.Fatal("explicit owned loopback endpoint required")
		}
		endpoints[name] = s
	}
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	type adapterState struct {
		Calls      int                 `json:"calls"`
		Unexpected int                 `json:"unexpected"`
		Inbox      []map[string]string `json:"inbox"`
	}
	state := func() adapterState {
		t.Helper()
		r, _ := http.NewRequest("GET", endpoints["FORMAL_ADAPTER_URL"]+"/_fixture/state", nil)
		r.Header.Set("X-Fixture-Key", "synthetic-fixture-control")
		resp, e := client.Do(r)
		if e != nil {
			t.Fatal("fixture state unavailable")
		}
		defer func() { _ = resp.Body.Close() }()
		var s adapterState
		if resp.StatusCode != 200 || json.NewDecoder(io.LimitReader(resp.Body, 65536)).Decode(&s) != nil || s.Unexpected != 0 {
			t.Fatal("invalid closed transport state")
		}
		return s
	}
	// Warden HTTP is a synthetic directory, not Warden runtime or migration proof.
	directory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			http.NotFound(w, r)
			return
		}
		mail := ""
		for _, vs := range r.URL.Query() {
			for _, v := range vs {
				if strings.HasSuffix(v, "@example.com") {
					mail = v
				}
			}
		}
		users := map[string][2]string{"a@example.com": {"user-A", "Ding-A_Z9"}, "b@example.com": {"user-B", "Ding-B_Z9"}, "u@example.com": {"user-U", ""}, "c@example.com": {"user-C", "12345678901"}}
		u, ok := users[mail]
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"user_id": u[0], "mail": mail, "phone": "13800138000", "dingtalk_userid": u[1], "status": "active"})
	}))
	defer directory.Close()
	setupSendVerifyCodeBaseEnv(t)
	for k, v := range map[string]string{"WARDEN_URL": directory.URL, "HERALD_URL": endpoints["FORMAL_HERALD_URL"], "HERALD_ENABLED": "true", "HERALD_API_KEY": "synthetic-service-key", "HERALD_HMAC_SECRET": "", "WARDEN_SOURCE_MAX_AGE": "0s", "LOGIN_DINGTALK_REQUIRE_USERID": "true", "LOGIN_SMS_ENABLED": "false", "LOGIN_EMAIL_ENABLED": "false", "HERALD_TOTP_ENABLED": "false"} {
		t.Setenv(k, v)
	}
	auth.ResetWardenClientForTesting()
	resetHeraldClientForTesting()
	old := getChallengeContextStore()
	bindings := newMemoryChallengeContextStore()
	SetChallengeContextStore(bindings)
	t.Cleanup(func() {
		auth.ResetWardenClientForTesting()
		resetHeraldClientForTesting()
		SetChallengeContextStore(old)
	})
	log := testLoggerSendVerifyCode()
	if config.Initialize(log) != nil || auth.InitWardenClient(log) != nil || InitHeraldClient(log) != nil {
		t.Fatal("native Stargate initialization failed")
	}
	app := fiber.New()
	store := setupTestStore()
	app.Post("/_send_verify_code", SendVerifyCodeAPI())
	app.Post("/_login", LoginAPI(store))
	app.Get("/_fixture/session", func(c fiber.Ctx) error {
		s, e := store.Get(c)
		if e != nil {
			return c.SendStatus(500)
		}
		defer s.Release()
		return c.JSON(fiber.Map{"authenticated": auth.IsAuthenticated(s), "user_id": auth.GetUserID(s)})
	})
	request := func(path string, form url.Values, key string) (int, map[string]any, []*http.Cookie) {
		t.Helper()
		r := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
		r.Host = "auth.example.com"
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Idempotency-Key", key)
		resp, e := app.Test(r, fiber.TestConfig{Timeout: 10 * time.Second})
		if e != nil {
			t.Fatal("native gateway request failed")
		}
		defer func() { _ = resp.Body.Close() }()
		var b map[string]any
		if json.NewDecoder(resp.Body).Decode(&b) != nil {
			t.Fatal("native gateway non-JSON response")
		}
		return resp.StatusCode, b, resp.Cookies()
	}
	send := url.Values{"mail": {"a@example.com"}, "deliver_via": {"dingtalk"}, "destination": {"Attacker"}, "to": {"Attacker"}, "dingtalk_userid": {"Attacker"}}
	status, created, _ := request("/_send_verify_code", send, "synthetic-three-A")
	id, _ := created["challenge_id"].(string)
	if status != 200 || created["success"] != true || id == "" {
		t.Fatal("native three-service creation failed")
	}
	t.Cleanup(func() { _ = bindings.Delete(context.Background(), id) })
	messages := state()
	if messages.Calls != 1 || len(messages.Inbox) != 1 {
		t.Fatal("expected one native upstream send")
	}
	code := messages.Inbox[0]["synthetic_code"]
	if len(code) != 6 {
		t.Fatal("missing native upstream synthetic OTP")
	}
	t.Run("opaque_mapping_and_client_injection", func(t *testing.T) {
		b, found, e := bindings.Get(context.Background(), id)
		if e != nil || !found || b.UserID != "user-A" || b.Destination != "Ding-A_Z9" || messages.Inbox[0]["to"] != "Ding-A_Z9" {
			t.Fatal("server-derived mapping changed")
		}
	})
	t.Run("same_request_idempotent", func(t *testing.T) {
		s, b, _ := request("/_send_verify_code", send, "synthetic-three-A")
		if s != 200 || b["challenge_id"] != id || state().Calls != 1 {
			t.Fatal("gateway replay repeated upstream delivery")
		}
	})
	t.Run("wrong_user_cannot_consume", func(t *testing.T) {
		s, b, _ := request("/_login", url.Values{"auth_method": {"warden"}, "mail": {"b@example.com"}, "challenge_id": {id}, "verify_code": {code}}, "")
		_, found, e := bindings.Get(context.Background(), id)
		if s != 401 || b["success"] == true || e != nil || !found {
			t.Fatal("cross-user attempt consumed owner challenge")
		}
	})
	t.Run("owner_session_and_consumed_replay", func(t *testing.T) {
		s, b, cookies := request("/_login", url.Values{"auth_method": {"warden"}, "mail": {"a@example.com"}, "challenge_id": {id}, "verify_code": {code}}, "")
		if s != 200 || b["success"] != true || len(cookies) == 0 {
			t.Fatal("owner native login failed")
		}
		r := httptest.NewRequest("GET", "/_fixture/session", nil)
		r.Host = "auth.example.com"
		for _, c := range cookies {
			r.AddCookie(c)
		}
		resp, e := app.Test(r)
		if e != nil {
			t.Fatal("native session read failed")
		}
		defer func() { _ = resp.Body.Close() }()
		var session map[string]any
		if json.NewDecoder(resp.Body).Decode(&session) != nil || session["authenticated"] != true || session["user_id"] != "user-A" {
			t.Fatal("native authenticated owner session absent")
		}
		s, b, _ = request("/_login", url.Values{"auth_method": {"warden"}, "mail": {"a@example.com"}, "challenge_id": {id}, "verify_code": {code}}, "")
		if s == 200 || b["success"] == true {
			t.Fatal("consumed challenge authenticated again")
		}
	})
	t.Run("unmapped_user_no_delivery", func(t *testing.T) {
		before := state().Calls
		s, b, _ := request("/_send_verify_code", url.Values{"mail": {"u@example.com"}, "deliver_via": {"dingtalk"}}, "synthetic-unmapped")
		if s != 400 || b["success"] == true || state().Calls != before {
			t.Fatal("strict mapping missing was not side-effect free")
		}
	})
	adapterSend := func(key, idem, to string) int {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"channel": "dingtalk", "to": to, "body": "Synthetic 654321"})
		r, _ := http.NewRequest("POST", endpoints["FORMAL_ADAPTER_URL"]+"/v1/send", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-API-Key", key)
		r.Header.Set("Idempotency-Key", idem)
		resp, e := client.Do(r)
		if e != nil {
			t.Fatal("native adapter request failed")
		}
		defer func() { _ = resp.Body.Close() }()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 65536))
		return resp.StatusCode
	}
	t.Run("adapter_auth_and_idempotency", func(t *testing.T) {
		before := state().Calls
		if adapterSend("", "synthetic-direct", "Ding-A_Z9") != 401 || adapterSend("wrong", "synthetic-direct", "Ding-A_Z9") != 401 || state().Calls != before {
			t.Fatal("adapter API auth failed")
		}
		if adapterSend("synthetic-adapter-key", "synthetic-direct", "Ding-A_Z9") != 200 || adapterSend("synthetic-adapter-key", "synthetic-direct", "Ding-A_Z9") != 200 || state().Calls != before+1 {
			t.Fatal("native adapter successful replay repeated send")
		}
		if adapterSend("synthetic-adapter-key", "synthetic-direct", "Ding-B_Z9") != 409 || state().Calls != before+1 {
			t.Fatal("native adapter fingerprint conflict failed")
		}
	})
	t.Run("provider_failure_refuses_login_challenge", func(t *testing.T) {
		r, _ := http.NewRequest("POST", endpoints["FORMAL_ADAPTER_URL"]+"/_fixture/fail", nil)
		r.Header.Set("X-Fixture-Key", "synthetic-fixture-control")
		resp, e := client.Do(r)
		if e != nil {
			t.Fatal("synthetic failure control failed")
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 204 {
			t.Fatal("synthetic failure control refused")
		}
		before := state().Calls
		s, b, _ := request("/_send_verify_code", url.Values{"mail": {"c@example.com"}, "deliver_via": {"dingtalk"}}, "synthetic-failure-C")
		if s == 200 || b["success"] == true || state().Calls <= before {
			t.Fatal("upstream failure did not fail closed")
		}
	})
}
