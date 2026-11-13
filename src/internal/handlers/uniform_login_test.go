package handlers

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/stretchr/testify/require"
)

// runReplaceLoginFailure serves one request whose handler first sets what a
// middleware would (a rate-limit header and a language cookie) and then runs
// attempt through replaceLoginFailure.
func runReplaceLoginFailure(t *testing.T, accept string, attempt func(ctx fiber.Ctx) (bool, bool, error)) (int, string, map[string][]string) {
	t.Helper()
	app := fiber.New()
	app.Post("/", func(ctx fiber.Ctx) error {
		ctx.Locals("i18n-bundle", i18n.GetBundle())
		ctx.Locals("i18n-language", i18n.LangEN)
		ctx.Set("X-RateLimit-Remaining", "9")
		ctx.Cookie(&fiber.Cookie{Name: "lang", Value: "en"})
		_, err := replaceLoginFailure(ctx, func() (bool, bool, error) { return attempt(ctx) })
		return err
	})
	request := httptest.NewRequest("POST", "/", nil)
	request.Header.Set("Accept", accept)
	response, err := app.Test(request, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	return response.StatusCode, string(body), response.Header
}

// A failure path may set anything on the response. None of it reaches the
// client, whatever format the client asked for; what was set before it does.
func TestReplaceLoginFailureDiscardsWhatTheFailurePathWrote(t *testing.T) {
	leak := func(ctx fiber.Ctx) (bool, bool, error) {
		ctx.Set("Retry-After", "30")
		ctx.Set("X-Account-State", "locked")
		ctx.Cookie(&fiber.Cookie{Name: "stargate_session_id", Value: "must-not-be-set"})
		return false, false, ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"reason": "locked", "user": "alice"})
	}
	for accept, want := range map[string]string{
		"application/json": uniformLoginFailureJSON,
		"application/xml":  `<errors><error code="401">` + uniformLoginFailure + `</error></errors>`,
		"text/plain":       uniformLoginFailure,
	} {
		t.Run(accept, func(t *testing.T) {
			status, body, headers := runReplaceLoginFailure(t, accept, leak)
			require.Equal(t, fiber.StatusUnauthorized, status)
			require.Equal(t, want, body)
			require.Equal(t, []string{accept}, headers["Content-Type"])
			require.NotContains(t, headers, "Retry-After")
			require.NotContains(t, headers, "X-Account-State")
			require.Equal(t, []string{"9"}, headers["X-Ratelimit-Remaining"], "headers set before the attempt are kept")
			require.Len(t, headers["Set-Cookie"], 1, "only the cookie set before the attempt is kept")
			require.Contains(t, headers["Set-Cookie"][0], "lang=en")
		})
	}
}

// A successful sign-in and a failed directory lookup keep their own answers.
func TestReplaceLoginFailureKeepsSuccessAndLookupFailure(t *testing.T) {
	for name, outcome := range map[string]struct{ authenticated, keep bool }{
		"authenticated":           {authenticated: true},
		"directory lookup failed": {keep: true},
	} {
		t.Run(name, func(t *testing.T) {
			status, body, headers := runReplaceLoginFailure(t, "application/json", func(ctx fiber.Ctx) (bool, bool, error) {
				ctx.Set("X-Kept", "yes")
				return outcome.authenticated, outcome.keep, ctx.Status(fiber.StatusServiceUnavailable).SendString("kept")
			})
			require.Equal(t, fiber.StatusServiceUnavailable, status)
			require.Equal(t, "kept", body)
			require.Equal(t, []string{"yes"}, headers["X-Kept"])
		})
	}
}
