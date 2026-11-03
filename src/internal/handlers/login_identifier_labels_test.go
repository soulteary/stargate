package handlers

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	views "github.com/gofiber/template/html/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"golang.org/x/net/html"
)

// Exercise the real login route and renderer so a label cannot silently select
// the other identifier field in screen readers or accessibility-based clients.
func TestWardenLoginIdentifierLabelsMatchTheirRenderedFields(t *testing.T) {
	old := config.WardenEnabled.Value
	config.WardenEnabled.Value = "true"
	t.Cleanup(func() { config.WardenEnabled.Value = old })
	for _, tc := range []struct {
		name         string
		lang         i18n.Language
		localized    bool
		phone, email string
	}{
		{"English", i18n.LangEN, true, "Phone number", "Email address"},
		{"Chinese", i18n.LangZH, true, "手机号", "邮箱地址"},
		{"English fallback without locale middleware", i18n.LangEN, false, "Phone number", "Email address"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{Views: views.New("../web/templates", ".html")})
			if tc.localized {
				app.Use(func(ctx fiber.Ctx) error {
					ctx.Locals("i18n-bundle", i18n.GetBundle())
					ctx.Locals("i18n-language", tc.lang)
					return ctx.Next()
				})
			}
			app.Get("/_login", LoginRoute(setupTestStore()))
			response, err := app.Test(httptest.NewRequest("GET", "http://auth.example.invalid/_login", nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != 200 {
				t.Fatalf("login route status %d", response.StatusCode)
			}
			document, err := html.Parse(io.LimitReader(response.Body, 1<<20))
			if err != nil {
				t.Fatal(err)
			}
			fields := map[string][]map[string]string{}
			labels := map[string][]string{}
			var text func(*html.Node) string
			text = func(n *html.Node) string {
				var b strings.Builder
				if n.Type == html.TextNode {
					b.WriteString(n.Data)
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					b.WriteString(text(c))
				}
				return b.String()
			}
			var visit func(*html.Node)
			visit = func(n *html.Node) {
				attrs := map[string]string{}
				for _, a := range n.Attr {
					attrs[a.Key] = a.Val
				}
				if n.Type == html.ElementNode && n.Data == "input" && (attrs["id"] == "phone" || attrs["id"] == "mail") {
					fields[attrs["id"]] = append(fields[attrs["id"]], attrs)
				}
				if n.Type == html.ElementNode && n.Data == "label" {
					labels[attrs["for"]] = append(labels[attrs["for"]], strings.TrimSpace(text(n)))
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					visit(c)
				}
			}
			visit(document)
			for _, field := range []struct{ id, label, kind string }{{"phone", tc.phone, "text"}, {"mail", tc.email, "email"}} {
				if len(fields[field.id]) != 1 || fields[field.id][0]["name"] != field.id || fields[field.id][0]["type"] != field.kind {
					t.Errorf("%s identifier input mapping changed", field.id)
				}
				if len(labels[field.id]) != 1 || labels[field.id][0] != field.label {
					t.Errorf("%s must have one correctly associated translated label; got %q", field.id, labels[field.id])
				}
			}
		})
	}
}
