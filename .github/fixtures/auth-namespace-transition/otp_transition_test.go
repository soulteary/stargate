package handlers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	challengekit "github.com/soulteary/challenge-kit"
	"github.com/soulteary/herald/internal/config"
)

// Native handler constructor selects its actual released challenge manager and
// prefixes. Direct manager operations deliberately exclude delivery/auth routing.
func TestNativeNamespaceOTP(t *testing.T) {
	root := os.Getenv("MIGRATION_ROOT")
	addr := os.Getenv("MIGRATION_REDIS")
	mode := os.Getenv("MIGRATION_MODE")
	if root == "" || !filepath.IsAbs(root) || !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatal("owned local fixture inputs required")
	}
	config.Env = config.EnvTest
	config.ChallengeExpiry = 300 * time.Second
	config.CodeLength = 6
	config.MaxAttempts = 5
	config.LockoutDuration = time.Hour
	config.IdempotencySecret = "synthetic-namespace-only-secret"
	config.PIIPepper = "synthetic-namespace-only-pepper"
	config.HeraldDingtalkAPIURL = ""
	config.HeraldSMTPAPIURL = ""
	config.SMTPHost = ""
	config.SMSProvider = ""
	config.SMSAPIBaseURL = ""
	config.TOTPEnabled = false
	rc := redis.NewClient(&redis.Options{Addr: addr})
	defer func() { _ = rc.Close() }()
	h, e := NewHandlersWithError(rc, testLogger())
	if e != nil {
		t.Fatal("native handler constructor failed")
	}
	defer func() {
		if h.StopAuditWriter() != nil {
			t.Error("native audit shutdown failed")
		}
	}()
	ctx := context.Background()
	type secret struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	}
	save := func(name string, v secret) {
		t.Helper()
		raw, _ := json.Marshal(v)
		f, e := os.OpenFile(filepath.Join(root, name+".private.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			t.Fatal("owned challenge create failed")
		}
		_, e = f.Write(raw)
		ce := f.Close()
		if e != nil || ce != nil {
			t.Fatal("owned challenge write failed")
		}
	}
	load := func(name string) secret {
		t.Helper()
		raw, e := os.ReadFile(filepath.Join(root, name+".private.json"))
		var v secret
		if e != nil || json.Unmarshal(raw, &v) != nil || v.ID == "" || len(v.Code) != 6 {
			t.Fatal("owned challenge missing")
		}
		return v
	}
	issue := func(name string) {
		t.Helper()
		ch, code, e := h.challengeManager.Create(ctx, challengekit.CreateRequest{UserID: "synthetic-" + name, Channel: challengekit.ChannelEmail, Destination: "synthetic@example.com", Purpose: "login", ClientIP: "127.0.0.1"})
		if e != nil {
			t.Fatal("native challenge create failed")
		}
		if mode == "new-issue" {
			if _, e := h.challengeManager.SwapActive(ctx, ch); e != nil {
				t.Fatal("native strict activation failed")
			}
		}
		save(name, secret{ch.ID, code})
	}
	verify := func(name string, want bool) {
		t.Helper()
		v := load(name)
		result, e := h.challengeManager.VerifyWithOptions(ctx, v.ID, v.Code, "127.0.0.1", challengekit.VerifyOptions{ExpectedUserID: "synthetic-" + name, ExpectedPurpose: "login", ExpectedChannel: challengekit.ChannelEmail})
		if result == nil || result.OK != want || (want && e != nil) || (!want && (e == nil || result.Reason != challengekit.ReasonExpired)) {
			t.Fatal("native OTP namespace result differs")
		}
	}
	switch mode {
	case "old-issue":
		issue("old-otp-witness")
		issue("old-otp-expiry")
		if rc.Set(ctx, "otp:lock:preserved-unrelated-user", "1", time.Hour).Err() != nil {
			t.Fatal("isolated lockout fixture seed failed")
		}
	case "new-reject-old":
		verify("old-otp-witness", false)
		verify("old-otp-expiry", false)
	case "unsafe-old-verifier-witness":
		verify("old-otp-witness", true)
	case "old-expired":
		verify("old-otp-expiry", false)
	case "new-issue":
		issue("new-otp")
	case "old-reject-new":
		verify("new-otp", false)
	case "new-verify":
		verify("new-otp", true)
		verify("new-otp", false)
	default:
		t.Fatal("unknown fixed OTP phase")
	}
	if rc.Get(ctx, "otp:lock:preserved-unrelated-user").Val() != "1" {
		t.Fatal("unrelated lockout was lost")
	}
}
