package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/handlers"
)

// Overlay is compiled separately inside each exact old/new Stargate module.
// Synthetic authenticated state is explicitly seeded through native auth helpers;
// this does not test credential proof, Warden, whole routing or browser exchange.
func TestNativeNamespaceSession(t *testing.T) {
	root := os.Getenv("MIGRATION_ROOT")
	addr := os.Getenv("MIGRATION_REDIS")
	mode := os.Getenv("MIGRATION_MODE")
	if root == "" || !filepath.IsAbs(root) || !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Fatal("owned local fixture inputs required")
	}
	prefix := "migration:ordinary:"
	if mode == "rollback-fresh" {
		prefix = "migration:rollback-fresh:"
	}
	for k, v := range map[string]string{"AUTH_HOST": "auth.example.com", "PASSWORDS": "plaintext:synthetic-unused", "SESSION_STORAGE_ENABLED": "true", "SESSION_STORAGE_REDIS_ADDR": addr, "SESSION_STORAGE_REDIS_DB": "0", "SESSION_STORAGE_REDIS_PASSWORD": "", "SESSION_STORAGE_REDIS_KEY_PREFIX": prefix, "COOKIE_SECURE": "false", "COOKIE_DOMAIN": "", "WARDEN_ENABLED": "false", "HERALD_ENABLED": "false", "HERALD_TOTP_ENABLED": "false", "AUDIT_LOG_ENABLED": "false"} {
		t.Setenv(k, v)
	}
	log = testLoggerMain()
	if config.Initialize(log) != nil {
		t.Fatal("native config initialization failed")
	}
	store, rc := setupSessionStore()
	if rc == nil {
		t.Fatal("native shared Redis session storage required")
	}
	defer func() { _ = rc.Close() }()
	app := fiber.New()
	app.Post("/_fixture/issue", func(c fiber.Ctx) error {
		s, e := store.Get(c)
		if e != nil {
			return c.SendStatus(500)
		}
		defer s.Release()
		auth.SetUserID(s, "synthetic-owner")
		if auth.Authenticate(s) != nil {
			return c.SendStatus(500)
		}
		return c.SendStatus(204)
	})
	app.Get("/_fixture/check", func(c fiber.Ctx) error {
		s, e := store.Get(c)
		if e != nil {
			return c.SendStatus(401)
		}
		defer s.Release()
		if !auth.IsAuthenticated(s) || auth.GetUserID(s) != "synthetic-owner" {
			return c.SendStatus(401)
		}
		return c.SendStatus(204)
	})
	app.Post("/_logout", handlers.LogoutRoute(store))
	path := func(name string) string { return filepath.Join(root, name+".private.json") }
	save := func(name, token string) {
		t.Helper()
		raw, _ := json.Marshal(map[string]string{"token": token})
		f, e := os.OpenFile(path(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			t.Fatal("owned token create failed")
		}
		_, e = f.Write(raw)
		ce := f.Close()
		if e != nil || ce != nil {
			t.Fatal("owned token write failed")
		}
	}
	load := func(name string) string {
		t.Helper()
		raw, e := os.ReadFile(path(name))
		if e != nil {
			t.Fatal("owned token missing")
		}
		var b map[string]string
		if json.Unmarshal(raw, &b) != nil || b["token"] == "" {
			t.Fatal("owned token shape invalid")
		}
		return b["token"]
	}
	request := func(method, p, token string) (int, string) {
		t.Helper()
		r := httptest.NewRequest(method, p, nil)
		r.Host = "auth.example.com"
		if token != "" {
			r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: token})
		}
		response, e := app.Test(r)
		if e != nil {
			t.Fatal("native session request failed")
		}
		defer func() { _ = response.Body.Close() }()
		value := ""
		for _, c := range response.Cookies() {
			if c.Name == auth.SessionCookieName && c.Value != "" {
				value = c.Value
			}
		}
		return response.StatusCode, value
	}
	check := func(name string, want int) {
		t.Helper()
		status, _ := request("GET", "/_fixture/check", load(name))
		if status != want {
			t.Fatal("native session namespace result differs")
		}
	}
	switch mode {
	case "old-issue":
		s, id := request("POST", "/_fixture/issue", "")
		if s != 204 || id == "" {
			t.Fatal("old native session issue failed")
		}
		save("old-session", id)
		check("old-session", 204)
	case "new-isolate":
		check("old-session", 401)
		s, id := request("POST", "/_fixture/issue", "")
		if s != 204 || id == "" {
			t.Fatal("new native session issue failed")
		}
		save("new-session", id)
		check("new-session", 204)
		if s, _ := request("POST", "/_logout", id); s != 200 {
			t.Fatal("native logout failed")
		}
		check("new-session", 401)
	case "unsafe-old-prefix-witness":
		check("old-session", 204)
		check("new-session", 401)
	case "rollback-fresh":
		check("old-session", 401)
		check("new-session", 401)
		s, id := request("POST", "/_fixture/issue", "")
		if s != 204 || id == "" {
			t.Fatal("fresh rollback login seed failed")
		}
		save("rollback-session", id)
		check("rollback-session", 204)
	case "new-after-rollback":
		check("old-session", 401)
		check("new-session", 401)
		check("rollback-session", 401)
	default:
		t.Fatal("unknown fixed session phase")
	}
}
