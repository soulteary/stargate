package handlers

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

const testExchangeNonce = "test-browser-nonce-at-least-32-bytes"

func testExchangeState(audience string) exchangeStateClaims {
	return exchangeStateClaims{Audience: normalizeExchangeAudience(audience), Challenge: exchangeChallenge(testExchangeNonce), RedirectPath: config.SessionExchangeRedirectPath.String(), Proto: "http", ExpiresAt: time.Now().Add(exchangeStateTTL).Unix()}
}
func bindTestExchangeBrowser(ctx fiber.Ctx) {
	ctx.Request().Header.SetCookie(exchangeCookieName(false), testExchangeNonce)
}
func bindTestExchangeRequest(t *testing.T, ctx fiber.Ctx, audience string) {
	t.Helper()
	token, err := sealExchangeValue(testExchangeState(audience), exchangeStateAAD)
	require.NoError(t, err)
	ctx.Request().Header.SetCookie(exchangeCookieName(true), token)
}

func TestExchangeStateAndTicketDomainsAndBrowserBindings(t *testing.T) {
	setSessionExchangeSecret(t)
	state := testExchangeState("app.example.com")
	token, err := sealExchangeValue(state, exchangeStateAAD)
	require.NoError(t, err)
	var decoded exchangeStateClaims
	require.NoError(t, openExchangeValue(token, exchangeStateAAD, &decoded))
	require.Equal(t, state, decoded)
	require.Error(t, openExchangeValue(token, exchangeTicketAAD, &decoded))
	for _, bad := range []string{"", "%%%", string(make([]byte, 8193))} {
		require.Error(t, openExchangeValue(bad, exchangeStateAAD, &decoded))
	}
	_, err = sealExchangeValue(make(chan int), exchangeStateAAD)
	require.Error(t, err)
	require.Empty(t, exchangeChallenge("short"))
	require.Empty(t, exchangeChallenge(string(make([]byte, 129))))
	require.False(t, validExchangeChallenge("%%%"))
	for _, mutate := range []func(*exchangeStateClaims){
		func(s *exchangeStateClaims) { s.Audience = "" },
		func(s *exchangeStateClaims) { s.Challenge = "bad" },
		func(s *exchangeStateClaims) { s.RedirectPath = "//evil.example" },
		func(s *exchangeStateClaims) { s.Proto = "ftp" },
		func(s *exchangeStateClaims) { s.ExpiresAt = time.Now().Add(-time.Second).Unix() },
	} {
		bad := state
		mutate(&bad)
		require.False(t, validExchangeState(bad))
		_, err = createSessionExchangeTicket("id", bad)
		require.Error(t, err)
	}
	for _, binding := range []struct{ nonce, proto, landing, audience string }{
		{"wrong-nonce-but-long-enough-to-hash", "http", state.RedirectPath, state.Audience},
		{testExchangeNonce, "https", state.RedirectPath, state.Audience},
		{testExchangeNonce, "http", "/different", state.Audience},
		{testExchangeNonce, "http", state.RedirectPath, "other.example.com"},
	} {
		ticket, err := createSessionExchangeTicket("id", state)
		require.NoError(t, err)
		_, err = consumeSessionExchangeTicket(ticket, binding.audience, binding.nonce, binding.proto, binding.landing)
		require.Error(t, err)
	}
	state.ExpiresAt = time.Now().Add(time.Second).Unix()
	ticket, err := createSessionExchangeTicket("id", state)
	require.NoError(t, err)
	var claims sessionExchangeClaims
	require.NoError(t, openExchangeValue(ticket, exchangeTicketAAD, &claims))
	require.Equal(t, state.ExpiresAt, claims.ExpiresAt)
}

func TestRequestedExchangeStateRejectsExplicitTamperingAndRestartsExpiredState(t *testing.T) {
	setSessionExchangeSecret(t)
	for _, tc := range []struct {
		name, token  string
		cookie, fail bool
	}{
		{name: "explicit malformed", token: "%%%", fail: true},
		{name: "cookie malformed", token: "%%%", cookie: true},
		{name: "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, app := createTestContext("GET", "/_login", nil, "")
			defer app.ReleaseCtx(ctx)
			if tc.cookie {
				ctx.Request().Header.SetCookie(exchangeCookieName(true), tc.token)
			} else {
				ctx.Request().URI().QueryArgs().Set(exchangeStateParameter, tc.token)
			}
			_, raw, err := requestedExchangeState(ctx, "app.example.com")
			require.Equal(t, tc.fail, err != nil)
			require.Empty(t, raw)
		})
	}
	for _, expired := range []bool{false, true} {
		state := testExchangeState("other.example.com")
		if expired {
			state.ExpiresAt = time.Now().Add(-time.Second).Unix()
		}
		token, err := sealExchangeValue(state, exchangeStateAAD)
		require.NoError(t, err)
		ctx, app := createTestContext("GET", "/_login?exchange_state="+token, nil, "")
		_, raw, err := requestedExchangeState(ctx, "app.example.com")
		require.Equal(t, !expired, err != nil)
		require.Empty(t, raw)
		app.ReleaseCtx(ctx)
		ctx, app = createTestContext("GET", "/_login", nil, "")
		ctx.Request().Header.SetCookie(exchangeCookieName(true), token)
		_, raw, err = requestedExchangeState(ctx, "app.example.com")
		require.NoError(t, err)
		require.Empty(t, raw)
		app.ReleaseCtx(ctx)
	}
}
