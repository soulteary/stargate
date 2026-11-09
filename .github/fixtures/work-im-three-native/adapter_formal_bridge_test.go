package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald-dingtalk/internal/config"
	"github.com/soulteary/logger-kit/v3"
)

type formalClosedTransport struct {
	mu                sync.Mutex
	calls, unexpected int
	fail              bool
	inbox             []map[string]string
}

func (c *formalClosedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	response := func(status int, body string) (*http.Response, error) {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}
	if r.URL.Scheme != "https" || r.URL.Host != "oapi.dingtalk.com" {
		c.unexpected++
		return nil, fmt.Errorf("closed transport rejected destination")
	}
	if r.Method == http.MethodGet && r.URL.Path == "/gettoken" {
		if r.URL.Query().Get("appkey") != "synthetic-app-key" || r.URL.Query().Get("appsecret") != "synthetic-app-secret" || len(r.URL.Query()) != 2 {
			c.unexpected++
			return nil, fmt.Errorf("closed token contract refused")
		}
		return response(200, `{"errcode":0,"access_token":"synthetic-token","expires_in":7200}`)
	}
	if r.Method != http.MethodPost || r.URL.Path != "/topapi/message/corpconversation/asyncsend_v2" || r.URL.Query().Get("access_token") != "synthetic-token" || len(r.URL.Query()) != 1 {
		c.unexpected++
		return nil, fmt.Errorf("closed transport rejected endpoint")
	}
	var b struct {
		Agent int64  `json:"agent_id"`
		To    string `json:"userid_list"`
		Msg   struct {
			Kind string `json:"msgtype"`
			Text struct {
				Content string `json:"content"`
			} `json:"text"`
		} `json:"msg"`
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 65537))
	if err != nil || len(raw) > 65536 || json.Unmarshal(raw, &b) != nil || b.Agent != 1 || b.Msg.Kind != "text" || (b.To != "Ding-A_Z9" && b.To != "Ding-B_Z9" && b.To != "12345678901") {
		c.unexpected++
		return nil, fmt.Errorf("closed send contract refused")
	}
	codes := regexp.MustCompile(`\b[0-9]{6}\b`).FindAllString(b.Msg.Text.Content, -1)
	if len(codes) != 1 {
		c.unexpected++
		return nil, fmt.Errorf("closed synthetic content refused")
	}
	c.calls++
	if c.fail {
		return response(503, `{"error":"synthetic_failure"}`)
	}
	c.inbox = append(c.inbox, map[string]string{"to": b.To, "synthetic_code": codes[0]})
	return response(200, `{"errcode":0,"task_id":123}`)
}
func TestFormalAdapterServe(t *testing.T) {
	ready := os.Getenv("FORMAL_ADAPTER_READY")
	if ready == "" {
		t.Fatal("explicit owned fixture readiness file required")
	}
	// Config package initialization only reads env; constructors perform no IO.
	// Override the only fallback transport BEFORE unchanged production Setup.
	closed := new(formalClosedTransport)
	old := http.DefaultTransport
	http.DefaultTransport = closed
	defer func() { http.DefaultTransport = old }()
	config.APIKey = "synthetic-adapter-key"
	config.AppKey = "synthetic-app-key"
	config.AppSecret = "synthetic-app-secret"
	config.AgentID = "1"
	config.LookupMode = config.LookupModeNone
	config.MaxConcurrentRequests = 32
	config.MaxRequestBodyBytes = 65536
	config.IdemTTLSec = 300
	app := fiber.New(fiber.Config{BodyLimit: config.EffectiveMaxRequestBodyBytes()})
	Setup(app, logger.New(logger.Config{Level: logger.ErrorLevel}))
	stop := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/_fixture/") {
			if r.Header.Get("X-Fixture-Key") != "synthetic-fixture-control" {
				w.WriteHeader(401)
				return
			}
			switch r.URL.Path {
			case "/_fixture/state":
				closed.mu.Lock()
				defer closed.mu.Unlock()
				_ = json.NewEncoder(w).Encode(map[string]any{"calls": closed.calls, "unexpected": closed.unexpected, "inbox": closed.inbox})
			case "/_fixture/fail":
				if r.Method != "POST" {
					w.WriteHeader(405)
					return
				}
				closed.mu.Lock()
				closed.fail = true
				closed.mu.Unlock()
				w.WriteHeader(204)
			case "/_fixture/stop":
				if r.Method != "POST" {
					w.WriteHeader(405)
					return
				}
				w.WriteHeader(204)
				once.Do(func() { close(stop) })
			default:
				w.WriteHeader(404)
			}
			return
		}
		response, err := app.Test(r, fiber.TestConfig{Timeout: 5 * time.Second})
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
	f, err := os.OpenFile(ready, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal("readiness create failed")
	}
	_, err = f.WriteString(server.URL)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		t.Fatal("readiness write failed")
	}
	select {
	case <-stop:
	case <-time.After(100 * time.Second):
		t.Fatal("owned adapter shutdown timeout")
	}
	closed.mu.Lock()
	defer closed.mu.Unlock()
	if closed.unexpected != 0 {
		t.Error("closed transport observed unexpected outbound request")
	}
}
