package handlers

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	views "github.com/gofiber/template/html/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// renderedLoginPage is what the login page hands to its script and shows as
// its hint.
type renderedLoginPage struct {
	messages         map[string]string
	singleIdentifier string
	hint             string
}

// loginPageMessages returns the message set the page script reads.
func loginPageMessages(t *testing.T, lang i18n.Language) map[string]string {
	t.Helper()
	return renderLoginPage(t, lang).messages
}

// renderLoginPage renders the login page through the real route and templates.
func renderLoginPage(t *testing.T, lang i18n.Language) renderedLoginPage {
	t.Helper()
	app := fiber.New(fiber.Config{Views: views.New("../web/templates", ".html")})
	app.Use(func(ctx fiber.Ctx) error {
		ctx.Locals("i18n-bundle", i18n.GetBundle())
		ctx.Locals("i18n-language", lang)
		return ctx.Next()
	})
	app.Get("/_login", LoginRoute(setupTestStore()))
	response, err := app.Test(httptest.NewRequest("GET", "http://auth.example.invalid/_login", nil), fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	require.NoError(t, err)
	defer func() { require.NoError(t, response.Body.Close()) }()
	require.Equal(t, fiber.StatusOK, response.StatusCode)
	document, err := html.Parse(io.LimitReader(response.Body, 1<<20))
	require.NoError(t, err)

	var encoded string
	page := renderedLoginPage{messages: map[string]string{}}
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		attributes := map[string]string{}
		for _, attribute := range node.Attr {
			attributes[attribute.Key] = attribute.Val
		}
		if attributes["id"] == "auth-ui-config" {
			encoded = attributes["data-messages"]
			page.singleIdentifier = attributes["data-single-identifier"]
		}
		if node.Type == html.ElementNode && node.Data == "p" && attributes["class"] == "hint" && page.hint == "" && node.FirstChild != nil {
			page.hint = node.FirstChild.Data
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	require.NotEmpty(t, encoded, "the login page must carry its message set")
	require.NoError(t, json.Unmarshal([]byte(encoded), &page.messages))
	return page
}

// With uniform responses the login page replaces the two phrases that would
// claim an outcome the server no longer discloses, and its hint asks for one
// identifier. Nothing else changes, and with the flag off the page is the one
// that exists today.
func TestUniformLoginPagePhrases(t *testing.T) {
	setConfigValue(t, &config.WardenEnabled, "true")
	setConfigValue(t, &config.HeraldEnabled, "true")
	setConfigValue(t, &config.LoginUniformResponses, "false")

	for _, lang := range []i18n.Language{i18n.LangEN, i18n.LangZH, i18n.LangFR, i18n.LangDE, i18n.LangIT, i18n.LangJA, i18n.LangKO} {
		t.Run(string(lang), func(t *testing.T) {
			original := i18n.AuthenticationUI(lang)
			replacement := i18n.UniformAuthenticationUI(lang)

			require.Len(t, replacement, 3)

			config.LoginUniformResponses.Value = "false"
			page := renderLoginPage(t, lang)
			require.Equal(t, original, page.messages)
			require.Equal(t, original["identifier_hint"], page.hint)
			require.Equal(t, "false", page.singleIdentifier)

			config.LoginUniformResponses.Value = "true"
			page = renderLoginPage(t, lang)
			require.Len(t, page.messages, len(original))
			for key, phrase := range original {
				if replaced, ok := replacement[key]; ok {
					require.Equal(t, replaced, page.messages[key], key)
					require.NotEqual(t, phrase, page.messages[key], key)
				} else {
					require.Equal(t, phrase, page.messages[key], key)
				}
			}
			// The page shows the hint and tells its script to ask for one identifier.
			require.Equal(t, replacement["identifier_hint"], page.hint)
			require.Equal(t, "true", page.singleIdentifier)
		})
	}
}

// Only the login page of a deployment that offers Warden sign-in through
// Herald changes. The other pages, and a login page without that sign-in,
// keep their texts.
func TestUniformPhrasesAreLimitedToTheWardenLoginPage(t *testing.T) {
	setConfigValue(t, &config.WardenEnabled, "true")
	setConfigValue(t, &config.HeraldEnabled, "true")
	setConfigValue(t, &config.LoginUniformResponses, "true")
	original := i18n.AuthenticationUI(i18n.LangEN)

	ctx, app := createTestContext("GET", "/_account", nil, "")
	defer app.ReleaseCtx(ctx)
	encoded, _ := authenticationView(ctx, "account")["UIMessages"].(string)
	messages := map[string]string{}
	require.NoError(t, json.Unmarshal([]byte(encoded), &messages))
	require.Equal(t, original, messages, "pages other than the login page keep the original phrases")

	config.HeraldEnabled.Value = "false"
	page := renderLoginPage(t, i18n.LangEN)
	require.Equal(t, original, page.messages, "no Herald: nothing to replace")
	require.Equal(t, "false", page.singleIdentifier, "no Herald: both identifiers are sent as before")
	config.HeraldEnabled.Value = "true"
	config.WardenEnabled.Value = "false"
	require.Equal(t, original, loginPageMessages(t, i18n.LangEN), "password-only login page")
}
