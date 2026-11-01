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

// Increment and expiry must commit together. PTTL distinguishes a live window
// with less than one second left from a counter that has no expiry.
var incrementRateLimit = redis.NewScript(`
local remaining = redis.call('PTTL', KEYS[1])
if remaining == -2 then
    redis.call('SET', KEYS[1], 1, 'PX', ARGV[1])
    return {1, tonumber(ARGV[1])}
end
if remaining < 0 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
    remaining = tonumber(ARGV[1])
end
local count = redis.call('INCR', KEYS[1])
return {count, remaining}
`)

func (s *redisRateLimitStore) Incr(ctx context.Context, key string, window time.Duration) (int, time.Duration, error) {
	milliseconds := window.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	result, err := incrementRateLimit.Run(ctx, s.client, []string{s.prefix + key}, milliseconds).Int64Slice()
	if err != nil {
		count, retryAfter, _ := s.fallback.Incr(ctx, key, window)
		return count, retryAfter, err
	}
	return int(result[0]), time.Duration(result[1]) * time.Millisecond, nil
}

var (
	rateLimitStoreMu sync.RWMutex
	rateLimitStore   RateLimitStore = newMemoryRateLimitStore()
)

// NewRateLimitStore shares rate-limit state through Redis when session storage
// provides a client, so every replica enforces one quota. Standalone
// deployments keep the process-local implementation.
func NewRateLimitStore(client redis.Cmdable) RateLimitStore {
	if redisClientMissing(client) {
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
