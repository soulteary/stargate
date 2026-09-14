package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/stargate/src/internal/config"
)

// RateLimitStore counts requests inside a fixed window. A shared
// implementation is required when Stargate runs more than one replica:
// a process-local counter gives every replica its own private quota, so the
// effective limit is the configured value multiplied by the replica count.
type RateLimitStore interface {
	// Incr records one request against key and returns the running count for
	// the current window together with the time remaining in that window.
	Incr(ctx context.Context, key string, window time.Duration) (int, time.Duration, error)
}

type rateLimitBucket struct {
	count  int
	resets time.Time
}

type memoryRateLimitStore struct {
	mu          sync.Mutex
	buckets     map[string]rateLimitBucket
	lastCleanup time.Time
}

func newMemoryRateLimitStore() *memoryRateLimitStore {
	return &memoryRateLimitStore{buckets: make(map[string]rateLimitBucket)}
}

func (s *memoryRateLimitStore) Incr(_ context.Context, key string, window time.Duration) (int, time.Duration, error) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Expired buckets are only reachable through their own key, so a periodic
	// sweep is enough to keep an idle process from retaining every address it
	// has ever seen.
	if s.lastCleanup.IsZero() || now.Sub(s.lastCleanup) >= window {
		for bucketKey, candidate := range s.buckets {
			if !now.Before(candidate.resets) {
				delete(s.buckets, bucketKey)
			}
		}
		s.lastCleanup = now
	}

	bucket := s.buckets[key]
	if bucket.resets.IsZero() || !now.Before(bucket.resets) {
		bucket = rateLimitBucket{resets: now.Add(window)}
	}
	bucket.count++
	s.buckets[key] = bucket

	return bucket.count, time.Until(bucket.resets), nil
}

// redisRateLimitStore shares one fixed-window counter across replicas.
//
// Redis errors fall back to the process-local store rather than failing open
// or failing closed: failing open would remove the limit entirely during a
// Redis outage, and failing closed would turn that outage into a total
// authentication outage. Degrading to per-replica limits keeps a bound in
// place without denying every request.
type redisRateLimitStore struct {
	client   redis.Cmdable
	prefix   string
	fallback *memoryRateLimitStore
}

func (s *redisRateLimitStore) Incr(ctx context.Context, key string, window time.Duration) (int, time.Duration, error) {
	redisKey := s.prefix + key

	// INCR and TTL travel together so the common path costs one round trip.
	pipeline := s.client.Pipeline()
	incr := pipeline.Incr(ctx, redisKey)
	ttl := pipeline.TTL(ctx, redisKey)
	if _, err := pipeline.Exec(ctx); err != nil {
		count, retryAfter, _ := s.fallback.Incr(ctx, key, window)
		return count, retryAfter, err
	}

	count := int(incr.Val())
	remaining := ttl.Val()
	// A counter without an expiry is either the first request of a window or a
	// key whose EXPIRE was lost; both need the window (re)armed, otherwise the
	// count would never reset.
	if remaining <= 0 {
		if err := s.client.Expire(ctx, redisKey, window).Err(); err != nil {
			return count, window, err
		}
		remaining = window
	}
	return count, remaining, nil
}

var (
	rateLimitStoreMu sync.RWMutex
	rateLimitStore   RateLimitStore = newMemoryRateLimitStore()
)

// NewRateLimitStore shares rate-limit state through Redis when session storage
// provides a client, so every replica enforces one quota. Standalone
// deployments keep the process-local implementation.
func NewRateLimitStore(client redis.Cmdable) RateLimitStore {
	if client == nil {
		return newMemoryRateLimitStore()
	}
	prefix := config.SessionStorageRedisKeyPrefix.String()
	if prefix == "" {
		prefix = "stargate:session:"
	}
	return &redisRateLimitStore{
		client:   client,
		prefix:   prefix + "ratelimit:",
		fallback: newMemoryRateLimitStore(),
	}
}

// SetRateLimitStore installs the store used by every rate-limited endpoint.
func SetRateLimitStore(store RateLimitStore) {
	rateLimitStoreMu.Lock()
	defer rateLimitStoreMu.Unlock()
	if store == nil {
		store = newMemoryRateLimitStore()
	}
	rateLimitStore = store
}

func getRateLimitStore() RateLimitStore {
	rateLimitStoreMu.RLock()
	defer rateLimitStoreMu.RUnlock()
	return rateLimitStore
}
