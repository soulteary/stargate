package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginQuotaUsesRegisteredRouteIdentity(t *testing.T) {
	oldMax, oldWindow := config.RateLimitLoginMax.Value, config.RateLimitWindow.Value
	t.Cleanup(func() {
		config.RateLimitLoginMax.Value = oldMax
		config.RateLimitWindow.Value = oldWindow
		resetRateLimitStateForTesting()
	})
	config.RateLimitLoginMax.Value = "1"
	config.RateLimitWindow.Value = "1m"
	resetRateLimitStateForTesting()
	app := fiber.New()
	app.Post("/_login", RequireSameOrigin(), LoginRateLimit(), LoginAPI(setupTestStore()))
	request := func(path string) int {
		r := httptest.NewRequest("POST", "http://auth.example.com"+path, strings.NewReader("password=wrong-password"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, err := app.Test(r)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		return response.StatusCode
	}
	require.Equal(t, 401, request("/_login"))
	require.Equal(t, 429, request("/_login"))
	require.Equal(t, 429, request("/_LOGIN"))
	require.Equal(t, 429, request("/_Login"))
	require.Equal(t, 429, request("/_lOgIn"))
	require.Equal(t, 429, request("/_login/"))
}
