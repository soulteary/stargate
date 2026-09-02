package handlers

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/stargate/src/internal/config"
)

const sessionExchangeTicketTTL = 2 * time.Minute

var consumedSessionExchangeTickets sync.Map

// SessionExchangeReplayStore atomically records a consumed ticket hash. A
// shared implementation is required when sessions are shared across replicas.
type SessionExchangeReplayStore interface {
	Consume(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

type memorySessionExchangeReplayStore struct {
	consumed *sync.Map
}

func (s *memorySessionExchangeReplayStore) Consume(_ context.Context, key string, ttl time.Duration) (bool, error) {
	if _, loaded := s.consumed.LoadOrStore(key, struct{}{}); loaded {
		return false, nil
	}
	time.AfterFunc(ttl, func() { s.consumed.Delete(key) })
	return true, nil
}

type redisSessionExchangeReplayStore struct {
	client redis.Cmdable
	prefix string
}

func (s *redisSessionExchangeReplayStore) Consume(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, s.prefix+key, "1", ttl).Result()
}

var defaultSessionExchangeReplayStore SessionExchangeReplayStore = &memorySessionExchangeReplayStore{
	consumed: &consumedSessionExchangeTickets,
}

// NewSessionExchangeReplayStore uses Redis when it is available so all
// Stargate replicas enforce the same single-use ticket state. Standalone
// in-memory deployments retain the process-local implementation.
func NewSessionExchangeReplayStore(client redis.Cmdable) SessionExchangeReplayStore {
	if client == nil {
		return defaultSessionExchangeReplayStore
	}
	prefix := config.SessionStorageRedisKeyPrefix.String()
	if prefix == "" {
		prefix = "stargate:session:"
	}
	return &redisSessionExchangeReplayStore{
		client: client,
		prefix: prefix + "exchange:used:",
	}
}

type sessionExchangeClaims struct {
	SessionID string `json:"sid"`
	Audience  string `json:"aud"`
	ExpiresAt int64  `json:"exp"`
}

func sessionExchangeAEAD() (cipher.AEAD, error) {
	secret := config.SessionExchangeSecret.String()
	if len(secret) < 32 {
		return nil, errors.New("session exchange is not configured")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func normalizeExchangeAudience(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\\\r\n\t") {
		return ""
	}
	value := raw
	if !strings.Contains(value, "://") {
		value = "//" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(parsed.Host, "."))
}

func createSessionExchangeTicket(sessionID, audience string) (string, error) {
	audience = normalizeExchangeAudience(audience)
	if sessionID == "" || audience == "" {
		return "", errors.New("invalid session exchange claims")
	}
	aead, err := sessionExchangeAEAD()
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(sessionExchangeClaims{
		SessionID: sessionID,
		Audience:  audience,
		ExpiresAt: time.Now().Add(sessionExchangeTicketTTL).Unix(),
	})
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nil, nonce, claims, nil)
	return base64.RawURLEncoding.EncodeToString(append(nonce, sealed...)), nil
}

//nolint:unused // Test helper; test packages are compiled and run separately.
func consumeSessionExchangeTicket(token, audience string) (string, error) {
	return consumeSessionExchangeTicketWithStore(context.Background(), token, audience, defaultSessionExchangeReplayStore)
}

func consumeSessionExchangeTicketWithStore(ctx context.Context, token, audience string, replayStore SessionExchangeReplayStore) (string, error) {
	audience = normalizeExchangeAudience(audience)
	if token == "" || audience == "" {
		return "", errors.New("invalid session exchange ticket")
	}
	aead, err := sessionExchangeAEAD()
	if err != nil {
		return "", err
	}
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(payload) <= aead.NonceSize() {
		return "", errors.New("invalid session exchange ticket")
	}
	claimsJSON, err := aead.Open(nil, payload[:aead.NonceSize()], payload[aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("invalid session exchange ticket")
	}
	var claims sessionExchangeClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return "", errors.New("invalid session exchange ticket")
	}
	remaining := time.Until(time.Unix(claims.ExpiresAt, 0))
	if claims.SessionID == "" || claims.Audience != audience || remaining <= 0 {
		return "", errors.New("expired or misdirected session exchange ticket")
	}

	ticketHash := sha256.Sum256([]byte(token))
	key := base64.RawURLEncoding.EncodeToString(ticketHash[:])
	if replayStore == nil {
		return "", errors.New("session exchange replay protection is not configured")
	}
	consumed, err := replayStore.Consume(ctx, key, remaining)
	if err != nil {
		return "", errors.New("failed to record session exchange ticket consumption")
	}
	if !consumed {
		return "", errors.New("session exchange ticket has already been used")
	}
	return claims.SessionID, nil
}
