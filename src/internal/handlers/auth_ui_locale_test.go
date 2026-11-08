package handlers

import (
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	i18nkit "github.com/soulteary/i18n-kit/v4"
	i18nfiber "github.com/soulteary/i18n-kit/v4/fiberadapter"
	"github.com/soulteary/stargate/src/internal/i18n"
	"net/http/httptest"
	"testing"
)

func TestAuthenticationUILocaleAndDeliveryShareResolvedLanguage(t *testing.T) {
	app := fiber.New()
	app.Use(i18nfiber.Middleware(i18nfiber.Config{MiddlewareConfig: i18nkit.MiddlewareConfig{Bundle: i18n.GetBundle()}}))
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"ui": authenticationUILanguage(c), "delivery": verificationLocale(c)})
	})
	req := httptest.NewRequest("GET", "/?lang=zh", nil)
	req.Header.Set("Cookie", "lang=en")
	req.Header.Set("X-Language", "en")
	req.Header.Set("Accept-Language", "en-US")
	r, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Body.Close() }()
	var body map[string]string
	if err = json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["ui"] != "zh" || body["delivery"] != "zh-CN" {
		t.Fatalf("split locale: %v", body)
	}
	ctx, testApp := createTestContext("GET", "/", nil, "")
	defer testApp.ReleaseCtx(ctx)
	ctx.Locals(i18nkit.LocalsLanguageKey, i18n.Language("unknown"))
	if authenticationUILanguage(ctx) != "en" {
		t.Fatal("unknown locale not normalized")
	}
}
