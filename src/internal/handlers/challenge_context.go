package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/stargate/src/internal/config"
)

type challengeContext struct {
	ChallengeID string `json:"challenge_id"`
	UserID      string `json:"user_id"`
	Channel     string `json:"channel"`
	Destination string `json:"destination"`
}

type ChallengeContextStore interface {
	PutIfAbsent(ctx context.Context, value challengeContext, ttl time.Duration) (bool, error)
	Get(ctx context.Context, challengeID string) (challengeContext, bool, error)
	Delete(ctx context.Context, challengeID string) error
}

// Keep trusted audit attribution briefly after Herald stops accepting a code so
// an expired verification response can still be recorded accurately.
const challengeContextAuditGrace = 5 * time.Minute

func challengeContextTTL(expiresIn int) time.Duration {
	return time.Duration(expiresIn)*time.Second + challengeContextAuditGrace
}

type memoryChallengeContextEntry struct {
	value     challengeContext
	expiresAt time.Time
	timer     *time.Timer
}

type memoryChallengeContextStore struct {
	mu      sync.Mutex
	entries map[string]memoryChallengeContextEntry
}

func newMemoryChallengeContextStore() *memoryChallengeContextStore {
	return &memoryChallengeContextStore{entries: make(map[string]memoryChallengeContextEntry)}
}

func (s *memoryChallengeContextStore) PutIfAbsent(_ context.Context, value challengeContext, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	if entry, ok := s.entries[value.ChallengeID]; ok && time.Now().Before(entry.expiresAt) {
		s.mu.Unlock()
		return false, nil
	} else if ok && entry.timer != nil {
		entry.timer.Stop()
	}
	expiresAt := time.Now().Add(ttl)
	challengeID := value.ChallengeID
	timer := time.AfterFunc(ttl, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if entry, ok := s.entries[challengeID]; ok && entry.expiresAt.Equal(expiresAt) {
			delete(s.entries, challengeID)
		}
	})
	s.entries[challengeID] = memoryChallengeContextEntry{value: value, expiresAt: expiresAt, timer: timer}
	s.mu.Unlock()
	return true, nil
}

func (s *memoryChallengeContextStore) Get(_ context.Context, challengeID string) (challengeContext, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[challengeID]
	if !ok {
		return challengeContext{}, false, nil
	}
	if !time.Now().Before(entry.expiresAt) {
		if entry.timer != nil {
			entry.timer.Stop()
		}
		delete(s.entries, challengeID)
		return challengeContext{}, false, nil
	}
	return entry.value, true, nil
}

func (s *memoryChallengeContextStore) Delete(_ context.Context, challengeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.entries[challengeID]; ok && entry.timer != nil {
		entry.timer.Stop()
	}
	delete(s.entries, challengeID)
	return nil
}

type redisChallengeContextStore struct {
	client redis.Cmdable
	prefix string
}

func (s *redisChallengeContextStore) PutIfAbsent(ctx context.Context, value challengeContext, ttl time.Duration) (bool, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return s.client.SetNX(ctx, s.prefix+value.ChallengeID, encoded, ttl).Result()
}

func (s *redisChallengeContextStore) Get(ctx context.Context, challengeID string) (challengeContext, bool, error) {
	encoded, err := s.client.Get(ctx, s.prefix+challengeID).Bytes()
	if err == redis.Nil {
		return challengeContext{}, false, nil
	}
	if err != nil {
		return challengeContext{}, false, err
	}
	var value challengeContext
	if err := json.Unmarshal(encoded, &value); err != nil {
		return challengeContext{}, false, err
	}
	return value, true, nil
}

func (s *redisChallengeContextStore) Delete(ctx context.Context, challengeID string) error {
	return s.client.Del(ctx, s.prefix+challengeID).Err()
}

var (
	challengeContextsMu sync.RWMutex
	challengeContexts   ChallengeContextStore = newMemoryChallengeContextStore()
)

func NewChallengeContextStore(client redis.Cmdable) ChallengeContextStore {
	if client == nil {
		return newMemoryChallengeContextStore()
	}
	prefix := config.SessionStorageRedisKeyPrefix.String()
	if prefix == "" {
		prefix = "stargate:session:"
	}
	return &redisChallengeContextStore{client: client, prefix: prefix + "challenge:context:"}
}

func SetChallengeContextStore(store ChallengeContextStore) {
	challengeContextsMu.Lock()
	defer challengeContextsMu.Unlock()
	if store == nil {
		store = newMemoryChallengeContextStore()
	}
	challengeContexts = store
}

func getChallengeContextStore() ChallengeContextStore {
	challengeContextsMu.RLock()
	defer challengeContextsMu.RUnlock()
	return challengeContexts
}

func loadChallengeContext(ctx context.Context, challengeID string) (challengeContext, error) {
	if challengeID == "" {
		return challengeContext{}, nil
	}
	value, found, err := getChallengeContextStore().Get(ctx, challengeID)
	if err != nil || !found {
		return challengeContext{}, err
	}
	return value, nil
}
