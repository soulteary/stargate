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
	// Reserve records one request against key like Incr and also returns a
	// function that takes that one request back. A caller that only wants to
	// count failures reserves before it evaluates the request and releases
	// when the request turns out to be good, so requests that are evaluated
	// at the same time cannot exceed the limit between them. Release is safe
	// to call once; it does nothing when the window has ended.
	Reserve(ctx context.Context, key string, window time.Duration) (count int, remaining time.Duration, release func(context.Context), err error)
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
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket := s.incrLocked(key, window)
	return bucket.count, time.Until(bucket.resets), nil
}

// incrLocked records one request against key and returns its bucket. The
// caller holds s.mu.
func (s *memoryRateLimitStore) incrLocked(key string, window time.Duration) rateLimitBucket {
	now := time.Now()

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
	return bucket
}

func (s *memoryRateLimitStore) Reserve(_ context.Context, key string, window time.Duration) (int, time.Duration, func(context.Context), error) {
	s.mu.Lock()
	reserved := s.incrLocked(key, window)
	s.mu.Unlock()

	release := func(context.Context) {
		s.mu.Lock()
		defer s.mu.Unlock()
		// A bucket with another reset time is a later window, which this
		// reservation was never counted in.
		bucket, ok := s.buckets[key]
		if !ok || !bucket.resets.Equal(reserved.resets) || bucket.count <= 0 {
			return
		}
		bucket.count--
		s.buckets[key] = bucket
	}
	return reserved.count, time.Until(reserved.resets), release, nil
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

// DECR keeps the expiry of the key. A missing key or a counter at zero is left
// alone: the window ended, and DECR would otherwise create a counter without
// an expiry.
const releaseRateLimitSource = `
local count = tonumber(redis.call('GET', KEYS[1]) or '0')
if count > 0 then
    return redis.call('DECR', KEYS[1])
end
return 0
`

var releaseRateLimit = redis.NewScript(releaseRateLimitSource)

// Reserve counts in Redis and releases there. When Redis cannot be reached the
// request is counted in the process-local store, like Incr, and released from
// the same place. A release that Redis does not accept is dropped: the request
// then stays counted until the window ends, which errs towards the limit.
func (s *redisRateLimitStore) Reserve(ctx context.Context, key string, window time.Duration) (int, time.Duration, func(context.Context), error) {
	milliseconds := window.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	result, err := s.increment(ctx, s.prefix+key, milliseconds)
	if err == nil && len(result) != 2 {
		err = errors.New("unexpected rate-limit script reply length")
	}
	if err != nil {
		count, remaining, release, _ := s.fallback.Reserve(ctx, key, window)
		return count, remaining, release, err
	}
	release := func(ctx context.Context) {
		_ = releaseRateLimit.Run(ctx, s.client, []string{s.prefix + key}).Err()
	}
	return int(result[0]), time.Duration(result[1]) * time.Millisecond, release, nil
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
