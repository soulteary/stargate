package handlers

import (
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/soulteary/stargate/src/internal/config"
)

func setSessionExchangeSecret(t *testing.T) {
	t.Helper()
	original := config.SessionExchangeSecret.Value
	t.Cleanup(func() { config.SessionExchangeSecret.Value = original })
	config.SessionExchangeSecret.Value = "test-session-exchange-secret-32-bytes"
}

func TestSessionExchangeTicketIsOpaqueAndAudienceBound(t *testing.T) {
	setSessionExchangeSecret(t)
	ticket, err := createSessionExchangeTicket("sensitive-session-id", "app.example.com")
	testza.AssertNoError(t, err)
	testza.AssertFalse(t, strings.Contains(ticket, "sensitive-session-id"))

	_, err = consumeSessionExchangeTicket(ticket, "other.example.com")
	testza.AssertNotNil(t, err)

	sessionID, err := consumeSessionExchangeTicket(ticket, "app.example.com")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "sensitive-session-id", sessionID)
}

func TestSessionExchangeTicketRejectsTamperingAndReplay(t *testing.T) {
	setSessionExchangeSecret(t)
	ticket, err := createSessionExchangeTicket("session-id", "app.example.com")
	testza.AssertNoError(t, err)

	_, err = consumeSessionExchangeTicket(ticket+"A", "app.example.com")
	testza.AssertNotNil(t, err)

	_, err = consumeSessionExchangeTicket(ticket, "app.example.com")
	testza.AssertNoError(t, err)
	_, err = consumeSessionExchangeTicket(ticket, "app.example.com")
	testza.AssertNotNil(t, err)
}
