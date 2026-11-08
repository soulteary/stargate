package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/html/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccountPageRequiresSessionAndLogoutRemainsSameOrigin(t *testing.T) {
	previous := config.AuthHost.Value
	config.AuthHost.Value = "auth.example"
	t.Cleanup(func() { config.AuthHost.Value = previous })
	store := setupTestStore()
	app := fiber.New(fiber.Config{Views: html.New("../web/templates", ".html")})
	app.Get("/_account", AccountRoute(store))
	app.Post("/_logout", RequireSameOrigin(), LogoutRoute(store))
	app.Get("/seed", func(c fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return err
		}
		defer sess.Release()
		if err = auth.Authenticate(sess); err != nil {
			return err
		}
		if err = sess.Save(); err != nil {
			return err
		}
		return c.SendStatus(200)
	})
	get := func(path string, cookies []*http.Cookie) *http.Response {
		req := httptest.NewRequest("GET", "http://auth.example"+path, nil)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		r, e := app.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	anonymous := get("/_account", nil)
	if anonymous.StatusCode != 302 || anonymous.Header.Get("Location") != "/_login" {
		t.Fatal("anonymous management access")
	}
	_ = anonymous.Body.Close()
	seed := get("/seed", nil)
	cookies := seed.Cookies()
	_ = seed.Body.Close()
	page := get("/_account", cookies)
	body, err := io.ReadAll(page.Body)
	_ = page.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if page.StatusCode != 200 || !strings.Contains(string(body), "<title>Account settings") || !strings.Contains(string(body), `method="post" action="/_logout"`) || page.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("management page: %d %s", page.StatusCode, body)
	}
	for _, origin := range []string{"https://outside.example", "http://auth.example"} {
		req := httptest.NewRequest("POST", "http://auth.example/_logout", nil)
		req.Header.Set("Origin", origin)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		r, e := app.Test(req)
		if e != nil {
			t.Fatal(e)
		}
		want := 403
		if origin == "http://auth.example" {
			want = 200
		}
		if r.StatusCode != want {
			t.Fatalf("logout origin result %d want%d", r.StatusCode, want)
		}
		_ = r.Body.Close()
	}
	after := get("/_account", cookies)
	defer func() { _ = after.Body.Close() }()
	if after.StatusCode != 302 {
		t.Fatalf("logout did not invalidate session: %d", after.StatusCode)
	}
}
