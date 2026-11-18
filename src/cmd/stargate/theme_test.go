package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/testsupport"
	"github.com/soulteary/stargate/src/internal/theme"
	"golang.org/x/net/html"
)

func testTheme(t *testing.T) *theme.Snapshot {
	t.Helper()
	dir := t.TempDir()
	raw := "contract: 1\nui_contract: 1\nid: test\nname: Test\nversion: 1.0.0\nauthor: Test\nlicense: MIT\nlayout: split\nbrand:\n  heading: '<script>alert(1)</script>'\n"
	if err := os.WriteFile(filepath.Join(dir, "theme.yml"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	selected, err := theme.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return selected
}

// The complete authentication subtree must be identical with and without a
// theme, including hidden state, conditional methods and secret controls.
func TestThemePreservesAuthenticationMarkup(t *testing.T) {
	isolateGlobalState(t)
	t.Chdir(t.TempDir())
	selected := testTheme(t)
	for _, lang := range []i18n.Language{i18n.LangEN, i18n.LangZH, i18n.LangFR, i18n.LangDE, i18n.LangIT, i18n.LangJA, i18n.LangKO} {
		for _, page := range []string{"login", "login.warden", "account", "totp_start", "totp_enroll", "totp_revoke", "step_up"} {
			for _, reauth := range []string{"password", "totp", "code"} {
				for _, enabled := range []bool{false, true} {
					words := i18n.AuthenticationUI(lang)
					encoded, err := json.Marshal(words)
					if err != nil {
						t.Fatal(err)
					}
					data := fiber.Map{"UI": words, "UIMessages": string(encoded), "UILanguage": string(lang), "Title": "Site <title>", "FooterText": "Footer", "SupportURL": "https://example.com/help", "Reauth": reauth, "HeraldEnabled": enabled, "OTPEnabled": enabled, "HeraldTOTPEnabled": enabled, "PasswordLoginEnabled": enabled, "LoginSMSEnabled": enabled, "LoginEmailEnabled": enabled, "SingleIdentifier": enabled, "Debug": enabled, "Callback": "app.example.com/path?a=1&b=2", "CallbackHost": "app.example.com", "CallbackProto": "https", "ExchangeState": "opaque-state", "EnrollID": "enrollment", "OtpauthURI": "otpauth://totp/Test?secret=TEST"}
					var normal, themed bytes.Buffer
					if err = setupThemeTemplates(nil, true).Render(&normal, page, data); err != nil {
						t.Fatal(page, err)
					}
					if err = setupThemeTemplates(selected, false).Render(&themed, page, data); err != nil {
						t.Fatal(page, err)
					}
					if authenticationMain(t, normal.String()) != authenticationMain(t, themed.String()) {
						t.Fatalf("authentication changed: %s %s %s %v", page, lang, reauth, enabled)
					}
					raw := themed.String()
					if !strings.Contains(raw, selected.BaseURL()+"/tokens.css") || !strings.Contains(raw, "auth-split") || strings.Contains(raw, "<script>alert(1)</script>") || !strings.Contains(raw, "&lt;script&gt;") {
						t.Fatalf("theme missing or unescaped: %s", page)
					}
					if _, ok := data["Theme"]; ok {
						t.Fatal("renderer mutated handler binding")
					}
				}
			}
		}
	}
}

func authenticationMain(t *testing.T, raw string) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	var main *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "main" {
			main = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if main == nil {
		t.Fatal("missing main")
	}
	var b bytes.Buffer
	if err = html.Render(&b, main); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestThemeUsesBuiltinsAndServesOnlySnapshot(t *testing.T) {
	setupTestConfig(t)
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("web/templates/assets", 0700); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"web/templates/login.html": "stale template", "web/templates/assets/auth-ui.js": "stale script", "web/templates/assets/favicon.ico": "stale icon"} {
		if err := os.WriteFile(name, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	selected := testTheme(t)
	for _, safe := range []bool{false, true} {
		active := selected
		if safe {
			active = nil
		}
		app := createAppWithTheme(active, safe)
		for _, target := range []string{"/_login", "/assets/auth-ui.js", "/assets/theme.css", "/favicon.ico", selected.BaseURL() + "/tokens.css", selected.BaseURL() + "/theme.yml", "/_theme/wrong/tokens.css"} {
			req := httptest.NewRequest("GET", target, nil)
			req.Host = "auth.example.com"
			resp, err := app.Test(req, testsupport.RequestAllowance)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			want := 200
			if strings.HasSuffix(target, "theme.yml") || strings.Contains(target, "wrong") || (safe && strings.HasPrefix(target, "/_theme/")) {
				want = 404
			}
			if resp.StatusCode != want {
				t.Fatalf("%s safe=%v status=%d body=%s", target, safe, resp.StatusCode, body)
			}
			if bytes.Contains(body, []byte("stale")) {
				t.Fatal("external override mixed into theme")
			}
			if target == "/_login" {
				if resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("Content-Security-Policy") != contentSecurityPolicy {
					t.Fatal("page protection changed")
				}
				if strings.Contains(string(body), "auth-themed") == safe {
					t.Fatal("safe mode selection")
				}
			}
			if !safe && strings.HasSuffix(target, "tokens.css") && resp.StatusCode == 200 {
				if resp.Header.Get("Content-Type") != "text/css; charset=utf-8" || resp.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
					t.Fatal("resource headers")
				}
			}
		}
	}
}

// TestThemeBrowserServer is an opt-in server for theme.py. Preview routes render
// synthetic view data only; password login and account/logout use real routes.
func TestThemeBrowserServer(t *testing.T) {
	directory := os.Getenv("STARGATE_THEME_BROWSER_DIR")
	if directory == "" {
		t.Skip("started by .github/browser/theme.py")
	}
	isolateGlobalState(t)
	t.Chdir(t.TempDir())
	for key, value := range map[string]string{"AUTH_HOST": os.Getenv("STARGATE_THEME_BROWSER_ADDR"), "PASSWORDS": "plaintext:" + os.Getenv("STARGATE_THEME_BROWSER_PASSWORD"), "COOKIE_SECURE": "false", "WARDEN_ENABLED": "false", "HERALD_ENABLED": "false", "AUDIT_LOG_ENABLED": "false", "LANGUAGE": "en"} {
		t.Setenv(key, value)
	}
	if err := initConfig(); err != nil {
		t.Fatal(err)
	}
	selected, err := theme.Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	app := createAppWithTheme(selected, false)
	t.Cleanup(func() { _ = app.Shutdown() })
	for _, name := range []string{"login", "login.warden", "account", "totp_start", "totp_enroll", "totp_revoke", "step_up"} {
		app.Get("/__theme_preview/"+name, func(c fiber.Ctx) error {
			lang := i18n.Language(c.Query("lang", "en"))
			words := i18n.AuthenticationUI(lang)
			encoded, encodeErr := json.Marshal(words)
			if encodeErr != nil {
				return encodeErr
			}
			c.Set("Cache-Control", "no-store")
			return c.Render(name, fiber.Map{"UI": words, "UIMessages": string(encoded), "UILanguage": string(lang), "Title": "Stargate", "FooterText": "Stargate", "Reauth": c.Query("reauth", "code"), "HeraldEnabled": true, "OTPEnabled": true, "HeraldTOTPEnabled": true, "PasswordLoginEnabled": true, "LoginSMSEnabled": true, "LoginEmailEnabled": true, "SingleIdentifier": c.Query("uniform") == "true", "Callback": "app.example.test", "ExchangeState": "preview-state", "EnrollID": "preview-enrollment", "OtpauthURI": "otpauth://totp/Preview?secret=JBSWY3DPEHPK3PXP"})
		})
	}
	if err = app.Listen(os.Getenv("STARGATE_THEME_BROWSER_ADDR")); err != nil {
		t.Fatal(err)
	}
}
