package handlers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/soulteary/stargate/src/internal/config"
)

func setSessionExchangeSecret(t *testing.T) {
	t.Helper()
	original := config.SessionExchangeSecret.Value
	t.Cleanup(func() { config.SessionExchangeSecret.Value = original })
	config.SessionExchangeSecret.Value = "test-session-exchange-secret-32-bytes"
}

func TestSessionExchangeTicketReplayIsSharedThroughRedis(t *testing.T) {
	setSessionExchangeSecret(t)
	redisClient := setupTestRedis(t)
	t.Cleanup(func() { _ = redisClient.Close() })

	firstReplica := NewSessionExchangeReplayStore(redisClient)
	secondReplica := NewSessionExchangeReplayStore(redisClient)
	ticket, err := createSessionExchangeTicket("shared-session-id", "app.example.com")
	testza.AssertNoError(t, err)

	sessionID, err := consumeSessionExchangeTicketWithStore(context.Background(), ticket, "app.example.com", firstReplica)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "shared-session-id", sessionID)

	_, err = consumeSessionExchangeTicketWithStore(context.Background(), ticket, "app.example.com", secondReplica)
	testza.AssertNotNil(t, err)
}

type failingSessionExchangeReplayStore struct{}

func (failingSessionExchangeReplayStore) Consume(context.Context, string, time.Duration) (bool, error) {
	return false, errors.New("backend unavailable")
}

func TestSessionExchangeTicketFailsClosedWhenReplayStoreFails(t *testing.T) {
	setSessionExchangeSecret(t)
	ticket, err := createSessionExchangeTicket("session-id", "app.example.com")
	testza.AssertNoError(t, err)

	_, err = consumeSessionExchangeTicketWithStore(context.Background(), ticket, "app.example.com", failingSessionExchangeReplayStore{})
	testza.AssertNotNil(t, err)
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
