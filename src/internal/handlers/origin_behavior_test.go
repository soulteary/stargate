package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func TestSameOriginMiddlewareAllowsSafeMethodsAndRejectsCrossSiteWrites(t *testing.T) {
	for _, tc := range []struct {
		method, site, origin string
		status               int
	}{
		{"GET", "cross-site", "", 204},
		{"HEAD", "cross-site", "", 204},
		{"OPTIONS", "cross-site", "", 204},
		{"POST", "cross-site", "", 403},
		{"POST", "same-origin", "", 204},
		{"POST", "same-origin", "https://other.example", 403},
	} {
		app := fiber.New()
		app.Use(RequireSameOrigin())
		app.Use(func(c fiber.Ctx) error { return c.SendStatus(204) })
		request := httptest.NewRequest(tc.method, "http://auth.example.com/", nil)
		request.Header.Set("Sec-Fetch-Site", tc.site)
		request.Header.Set("Origin", tc.origin)
		response, err := app.Test(request)
		require.NoError(t, err)
		require.Equal(t, tc.status, response.StatusCode)
		require.NoError(t, response.Body.Close())
	}
}
