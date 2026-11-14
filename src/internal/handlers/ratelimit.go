package handlers

import (
	"context"
	"errors"
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
	// Count returns the running count for key and the time remaining in its
	// window without recording a request. A key with no live window counts
	// zero.
	Count(ctx context.Context, key string) (int, time.Duration, error)
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

func (s *memoryRateLimitStore) Count(_ context.Context, key string) (int, time.Duration, error) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, ok := s.buckets[key]
	if !ok || !now.Before(bucket.resets) {
		return 0, 0, nil
	}
	return bucket.count, bucket.resets.Sub(now), nil
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
const incrementRateLimitSource = `
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
`

var incrementRateLimit = redis.NewScript(incrementRateLimitSource)

func (s *redisRateLimitStore) Incr(ctx context.Context, key string, window time.Duration) (int, time.Duration, error) {
	milliseconds := window.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	result, err := s.increment(ctx, s.prefix+key, milliseconds)
	if err == nil && len(result) != 2 {
		err = errors.New("unexpected rate-limit script reply length")
	}
	if err != nil {
		count, retryAfter, _ := s.fallback.Incr(ctx, key, window)
		return count, retryAfter, err
	}
	return int(result[0]), time.Duration(result[1]) * time.Millisecond, nil
}

// Count reads the shared counter. Like Incr it answers from the process-local
// store when Redis cannot be read, so an outage neither hides the failures
// counted locally nor rejects every request.
func (s *redisRateLimitStore) Count(ctx context.Context, key string) (int, time.Duration, error) {
	count, err := s.client.Get(ctx, s.prefix+key).Int()
	if errors.Is(err, redis.Nil) {
		return 0, 0, nil
	}
	var remaining time.Duration
	if err == nil {
		remaining, err = s.client.PTTL(ctx, s.prefix+key).Result()
	}
	if err != nil {
		count, remaining, _ = s.fallback.Count(ctx, key)
		return count, remaining, err
	}
	if remaining < 0 {
		// The key expired between the two commands or carries no expiry yet.
		remaining = 0
	}
	return count, remaining, nil
}

// A typed reply avoids generic RESP integers and a second slice conversion.
// Custom adapters retain the standard Script.Run path and their overrides.
func (s *redisRateLimitStore) increment(ctx context.Context, key string, milliseconds int64) ([]int64, error) {
	var processor interface {
		Process(context.Context, redis.Cmder) error
	}
	switch client := s.client.(type) {
	case *redis.Client:
		processor = client
	case *redis.ClusterClient:
		processor = client
	case *redis.Ring:
		processor = client
	default:
		// Adapters can override Eval/EvalSha for policy or instrumentation. Do not
		// bypass those methods even if a wrapped client's Process is promoted.
		return incrementRateLimit.Run(ctx, s.client, []string{key}, milliseconds).Int64Slice()
	}
	cmd := redis.NewIntSliceCmd(ctx, "evalsha", incrementRateLimit.Hash(), 1, key, milliseconds)
	cmd.SetFirstKeyPos(3)
	if err := processor.Process(ctx, cmd); err != nil {
		if !errors.Is(err, redis.ErrNoScript) && !redis.HasErrorPrefix(err, "NOSCRIPT") {
			return nil, err
		}
		// NOSCRIPT proves the operation did not run. No application retry is made
		// for timeouts, lost acknowledgements, or other uncertain outcomes.
		cmd = redis.NewIntSliceCmd(ctx, "eval", incrementRateLimitSource, 1, key, milliseconds)
		cmd.SetFirstKeyPos(3)
		if err := processor.Process(ctx, cmd); err != nil {
			return nil, err
		}
	}
	return cmd.Result()
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
