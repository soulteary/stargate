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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
)

// The separate Herald executable is built from the explicitly selected source.
// Its provider is a local synthetic inbox. No DingTalk credentials or real
// delivery are used. Existing unit tests cover source lookup and context rules;
// this test connects their actual handlers and released HTTP client contracts.
func TestWorkIMCurrentPair(t *testing.T) {
	bin := os.Getenv("WORK_IM_HERALD_TEST_BINARY")
	if bin == "" {
		t.Skip("explicit source-pinned Herald local bridge executable required")
	}
	file := filepath.Join(t.TempDir(), "fixture-url")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, bin, "-test.run=^TestWorkIMPilotServeLocalBridge$", "-test.timeout=48s")
	cmd.Env = append(os.Environ(), "WORK_IM_FIXTURE_URL_FILE="+file)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var bridge string
	for i := 0; i < 200; i++ {
		if b, err := os.ReadFile(file); err == nil {
			bridge = string(b)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if bridge == "" {
		cancel()
		_ = cmd.Wait()
		t.Fatal("selected Herald bridge did not become ready")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	t.Cleanup(func() {
		r, err := client.Post(bridge+"/_test/shutdown", "application/json", nil)
		if err != nil {
			t.Errorf("local fixture shutdown request failed: %v", err)
		} else {
			_ = r.Body.Close()
			if r.StatusCode != http.StatusNoContent {
				t.Errorf("shutdown status=%d", r.StatusCode)
			}
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("source-pinned fixture did not exit cleanly: %v", err)
		}
	})
	directory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			http.NotFound(w, r)
			return
		}
		user, mail, receiver := "user-A", "a@example.com", "Ding-A_Z9"
		for _, values := range r.URL.Query() {
			for _, value := range values {
				if value == "b@example.com" {
					user, mail, receiver = "user-B", "b@example.com", "Ding-B_Z9"
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"user_id": user, "mail": mail, "dingtalk_userid": receiver, "status": "active"})
	}))
	defer directory.Close()
	setupSendVerifyCodeBaseEnv(t)
	t.Setenv("WARDEN_URL", directory.URL)
	t.Setenv("HERALD_URL", bridge)
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_API_KEY", "synthetic-service-key")
	t.Setenv("HERALD_HMAC_SECRET", "")
	t.Setenv("WARDEN_SOURCE_MAX_AGE", "")
	auth.ResetWardenClientForTesting()
	resetHeraldClientForTesting()
	previousBindings := getChallengeContextStore()
	bindings := newMemoryChallengeContextStore()
	SetChallengeContextStore(bindings)
	t.Cleanup(func() {
		auth.ResetWardenClientForTesting()
		resetHeraldClientForTesting()
		SetChallengeContextStore(previousBindings)
	})
	log := testLoggerSendVerifyCode()
	if err := config.Initialize(log); err != nil {
		t.Fatal(err)
	}
	if err := auth.InitWardenClient(log); err != nil {
		t.Fatal(err)
	}
	if err := InitHeraldClient(log); err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	store := setupTestStore()
	app.Post("/_send_verify_code", SendVerifyCodeAPI())
	app.Post("/_login", LoginAPI(store))
	request := func(path string, form url.Values, key string) (int, map[string]any) {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		r.Host = "auth.example.com"
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Idempotency-Key", key)
		resp, err := app.Test(r, fiber.TestConfig{Timeout: 10 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal("non-JSON gateway response")
		}
		return resp.StatusCode, body
	}
	inbox := func() []map[string]string {
		t.Helper()
		r, err := client.Get(bridge + "/_test/inbox")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = r.Body.Close() }()
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var messages []map[string]string
		if err := json.Unmarshal(b, &messages); err != nil {
			t.Fatal(err)
		}
		return messages
	}
	send := url.Values{"mail": {"a@example.com"}, "deliver_via": {"dingtalk"}, "destination": {"Attacker"}, "to": {"Attacker"}, "dingtalk_userid": {"Attacker"}}
	status, created := request("/_send_verify_code", send, "synthetic-current-pair-A")
	if status != 200 || created["success"] != true {
		t.Fatalf("actual challenge creation status=%d", status)
	}
	id, ok := created["challenge_id"].(string)
	if !ok || id == "" {
		t.Fatal("missing committed challenge ID")
	}
	t.Cleanup(func() { _ = bindings.Delete(context.Background(), id) })
	messages := inbox()
	t.Run("opaque receiver survives both actual services", func(t *testing.T) {
		if len(messages) != 1 || messages[0]["channel"] != "dingtalk" || messages[0]["to"] != "Ding-A_Z9" || messages[0]["challenge_id"] != id {
			t.Fatal("trusted opaque receiver or challenge changed before controlled provider")
		}
		binding, found, err := bindings.Get(context.Background(), id)
		if err != nil || !found || binding.UserID != "user-A" || binding.Destination != "Ding-A_Z9" {
			t.Fatal("gateway durable binding differs from actual delivery")
		}
	})
	t.Run("same request replays without another delivery", func(t *testing.T) {
		status, replayed := request("/_send_verify_code", send, "synthetic-current-pair-A")
		if status != 200 || replayed["challenge_id"] != id || len(inbox()) != 1 {
			t.Fatal("idempotent replay changed challenge or repeated send")
		}
	})
	code := ""
	if len(messages) == 1 {
		code = messages[0]["synthetic_code"]
	}
	if len(code) != 6 {
		t.Fatal("controlled provider did not receive one synthetic code")
	}
	t.Run("other user cannot redeem the challenge", func(t *testing.T) {
		status, body := request("/_login", url.Values{"auth_method": {"warden"}, "mail": {"b@example.com"}, "challenge_id": {id}, "verify_code": {code}}, "")
		if status != 401 || body["success"] == true {
			t.Fatalf("cross-user challenge status=%d", status)
		}
	})
	t.Run("owner can still redeem after refused cross-user attempt", func(t *testing.T) {
		status, body := request("/_login", url.Values{"auth_method": {"warden"}, "mail": {"a@example.com"}, "challenge_id": {id}, "verify_code": {code}}, "")
		if status != 200 || body["success"] != true {
			t.Fatalf("owner login status=%d", status)
		}
	})
	t.Run("consumed challenge cannot authenticate again", func(t *testing.T) {
		status, body := request("/_login", url.Values{"auth_method": {"warden"}, "mail": {"a@example.com"}, "challenge_id": {id}, "verify_code": {code}}, "")
		if status == 200 || body["success"] == true {
			t.Fatalf("consumed challenge replay status=%d", status)
		}
	})
}
