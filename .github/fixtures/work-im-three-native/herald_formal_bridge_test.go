package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/herald/internal/config"
)

func TestFormalHeraldServe(t *testing.T) {
	ready := os.Getenv("FORMAL_HERALD_READY")
	endpoint := os.Getenv("FORMAL_ADAPTER_URL")
	u, err := url.Parse(endpoint)
	if ready == "" || err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.Path != "" || u.RawQuery != "" || u.User != nil {
		t.Fatal("owned loopback adapter required")
	}
	config.Env = config.EnvTest
	config.RequestAuthMode = "api_key"
	config.APIKey = "synthetic-service-key"
	config.HMACSecret = ""
	config.HMACKeysJSON = ""
	config.ServicePoliciesJSON = ""
	config.HeraldDingtalkAPIURL = endpoint
	config.HeraldDingtalkAPIKey = "synthetic-adapter-key"
	config.HeraldSMTPAPIURL = ""
	config.SMTPHost = ""
	config.SMSProvider = ""
	config.SMSAPIBaseURL = ""
	config.TOTPEnabled = false
	config.IdempotencySecret = "synthetic-idempotency-secret"
	config.PIIPepper = "synthetic-pii-pepper"
	config.ProviderFailurePolicy = "strict"
	config.TestMode = false
	config.AuditSpoolDir = t.TempDir()
	config.RateLimitPerIP = 5
	config.RateLimitPerUser = 10
	config.RateLimitPerDestination = 10
	config.ResendCooldown = time.Second
	config.ChallengeExpiry = 5 * time.Minute
	config.ProviderTimeout = 3 * time.Second
	config.CodeLength = 6
	config.MaxAttempts = 5
	config.AllowedPurposes = []string{"login"}
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal("synthetic Redis startup failed")
	}
	defer mr.Close()
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rc.Close() }()
	native, err := NewRouterWithClientAndHandlersE(rc, testLogger())
	if err != nil {
		t.Fatal("native Herald router construction failed")
	}
	defer func() {
		if err := native.Handlers.StopAuditWriter(); err != nil {
			t.Error("native audit shutdown failed")
		}
	}()
	stop := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_fixture/stop" {
			if r.Method != "POST" || r.Header.Get("X-Fixture-Key") != "synthetic-fixture-control" {
				w.WriteHeader(401)
				return
			}
			w.WriteHeader(204)
			once.Do(func() { close(stop) })
			return
		}
		response, err := native.App.Test(r)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		defer func() { _ = response.Body.Close() }()
		for k, vs := range response.Header {
			for _, x := range vs {
				w.Header().Add(k, x)
			}
		}
		w.WriteHeader(response.StatusCode)
		_, _ = io.Copy(w, response.Body)
	}))
	defer server.Close()
	if strings.Contains(endpoint, "@") {
		t.Fatal("userinfo prohibited")
	}
	// The runner must never observe a partial readiness endpoint. Both paths
	// belong to its freshly created private directory; no other publisher exists.
	pending := ready + ".pending"
	f, err := os.OpenFile(pending, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal("readiness create failed")
	}
	_, err = f.WriteString(server.URL)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		t.Fatal("readiness write failed")
	}
	if err := os.Rename(pending, ready); err != nil {
		t.Fatal("readiness publish failed")
	}
	select {
	case <-stop:
	case <-time.After(90 * time.Second):
		t.Fatal("owned Herald shutdown timeout")
	}
}
