package handlers

import (
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"io"
	"net/http/httptest"
	"testing"
)

func TestAuthenticationErrorPreservesExistingWireFormats(t *testing.T) {
	for _, accept := range []string{"application/json", "application/xml", "text/plain"} {
		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error { return SendAuthenticationError(c, 401, "Existing message", "invalid") })
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept", accept)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 401 {
			t.Fatalf("status=%d", resp.StatusCode)
		}
		if accept == "application/json" {
			var got map[string]any
			if err = json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}
			if got["code"] != float64(401) || got["error"] != "Existing message" || got["reason"] != "invalid" {
				t.Fatalf("body=%s", b)
			}
		}
	}
}

func TestAuthenticationReasonRejectsUntrustedDetail(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		return SendAuthenticationError(c, 401, "Existing message", "upstream-private-account-detail")
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var got map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["reason"] != "authentication_failed" {
		t.Fatalf("unexpected reason: %v", got)
	}
}
