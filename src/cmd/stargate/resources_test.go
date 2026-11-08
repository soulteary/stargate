package main

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestEmbeddedResourcesFallbackAndExternalOverrides(t *testing.T) {
	log = testLoggerMain()
	t.Chdir(t.TempDir())
	app := fiber.New(fiber.Config{Views: setupTemplates()})
	setupStaticFiles(app)
	app.Get("/page", func(c fiber.Ctx) error {
		return c.Render("login", fiber.Map{"UI": map[string]string{"signin": "Sign in"}})
	})
	for _, target := range []string{"/page", "/assets/auth-ui.js", "/assets/auth-ui.css", "/assets/qrcode.min.js"} {
		r, err := app.Test(httptest.NewRequest("GET", target, nil))
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil || r.StatusCode != 200 || len(body) == 0 {
			t.Fatalf("resource %s: status%d err%v", target, r.StatusCode, err)
		}
	}
	r, err := app.Test(httptest.NewRequest("GET", "/assets/account.html", nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Body.Close()
	if r.StatusCode == 200 {
		t.Fatal("template exposed through assets")
	}
	if err = os.MkdirAll(filepath.Join("web", "templates", "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join("web", "templates", "login.html"), []byte("External template"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join("web", "templates", "assets", "auth-ui.js"), []byte("External script"), 0600); err != nil {
		t.Fatal(err)
	}
	external := fiber.New(fiber.Config{Views: setupTemplates()})
	setupStaticFiles(external)
	external.Get("/page", func(c fiber.Ctx) error { return c.Render("login", fiber.Map{}) })
	for _, tc := range []struct{ path, want string }{{"/page", "External template"}, {"/assets/auth-ui.js", "External script"}} {
		response, e := external.Test(httptest.NewRequest("GET", tc.path, nil))
		if e != nil {
			t.Fatal(e)
		}
		body, e := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if e != nil || !strings.Contains(string(body), tc.want) {
			t.Fatalf("override %s not used: %q", tc.path, body)
		}
	}
}
