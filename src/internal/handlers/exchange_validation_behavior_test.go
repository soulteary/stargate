package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestExchangeRejectsMalformedAndExpiredClaims(t *testing.T) {
	setSessionExchangeSecret(t)
	for _, audience := range []string{"", "https://user@host", "app.example/path", "app.example?query=1", "app.example#fragment", "bad\t.example", "http://%"} {
		require.Empty(t, normalizeExchangeAudience(audience))
	}
	for _, token := range []string{"", "%%%", base64.RawURLEncoding.EncodeToString([]byte("short"))} {
		_, err := consumeSessionExchangeTicket(token, "app.example.com")
		require.Error(t, err)
	}
	_, err := createSessionExchangeTicket("", "app.example.com")
	require.Error(t, err)
	aead, err := sessionExchangeAEAD()
	require.NoError(t, err)
	for _, payload := range [][]byte{
		[]byte("invalid JSON"),
		mustExchangeClaims(t, sessionExchangeClaims{SessionID: "session", Audience: "app.example.com", ExpiresAt: time.Now().Add(-time.Minute).Unix()}),
		mustExchangeClaims(t, sessionExchangeClaims{Audience: "app.example.com", ExpiresAt: time.Now().Add(time.Minute).Unix()}),
	} {
		nonce := make([]byte, aead.NonceSize())
		token := base64.RawURLEncoding.EncodeToString(aead.Seal(nonce, nonce, payload, nil))
		_, err := consumeSessionExchangeTicket(token, "app.example.com")
		require.Error(t, err)
	}
	ticket, err := createSessionExchangeTicket("session", "app.example.com")
	require.NoError(t, err)
	_, err = consumeSessionExchangeTicketWithStore(context.Background(), ticket, "app.example.com", nil)
	require.ErrorContains(t, err, "not configured")
	config.SessionExchangeSecret.Value = "short"
	_, err = createSessionExchangeTicket("session", "app.example.com")
	require.Error(t, err)
	_, err = consumeSessionExchangeTicket(ticket, "app.example.com")
	require.Error(t, err)
}

func mustExchangeClaims(t *testing.T, claims sessionExchangeClaims) []byte {
	t.Helper()
	data, err := json.Marshal(claims)
	require.NoError(t, err)
	return data
}
