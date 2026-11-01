package handlers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type loseRateLimitReply struct{ once sync.Once }

func (*loseRateLimitReply) DialHook(next redis.DialHook) redis.DialHook { return next }
func (*loseRateLimitReply) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *loseRateLimitReply) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if err == nil && (cmd.Name() == "eval" || cmd.Name() == "evalsha") {
			h.once.Do(func() { err = errors.New("injected lost script acknowledgement") })
		}
		return err
	}
}

func TestRedisRateLimitRealCommittedWriteSurvivesLostReply(t *testing.T) {
	store := realRateLimitStore(t)
	store.client.(*redis.Client).AddHook(&loseRateLimitReply{})
	ctx := context.Background()
	if _, _, err := store.Incr(ctx, "lost-reply", time.Minute); err == nil {
		t.Fatal("lost acknowledgement must remain observable")
	}
	key := store.prefix + "lost-reply"
	if value, err := store.client.Get(ctx, key).Result(); err != nil || value != "1" {
		t.Fatalf("committed counter lost: value=%q err=%v", value, err)
	}
	if ttl := store.client.PTTL(ctx, key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("committed counter has invalid expiry: %s", ttl)
	}
	if count, _, err := store.Incr(ctx, "lost-reply", time.Minute); err != nil || count != 2 {
		t.Fatalf("retry discarded the committed count: count=%d err=%v", count, err)
	}
}

func realRateLimitStore(t testing.TB) *redisRateLimitStore {
	t.Helper()
	addr := os.Getenv("STARGATE_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("STARGATE_TEST_REDIS_ADDR is not configured")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	return &redisRateLimitStore{client: client, prefix: fmt.Sprintf("test:%s:%d:", t.Name(), time.Now().UnixNano()), fallback: newMemoryRateLimitStore()}
}

func TestRedisRateLimitRealConcurrentWindow(t *testing.T) {
	store := realRateLimitStore(t)
	ctx := context.Background()
	const requests = 64
	counts := make(chan int, requests)
	var workers sync.WaitGroup
	for i := 0; i < requests; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			count, ttl, err := store.Incr(ctx, "client", time.Second)
			if err != nil || ttl <= 0 || ttl > time.Second {
				t.Errorf("count=%d ttl=%s err=%v", count, ttl, err)
			}
			counts <- count
		}()
	}
	workers.Wait()
	close(counts)
	seen := make(map[int]bool)
	for count := range counts {
		if count < 1 || count > requests || seen[count] {
			t.Errorf("invalid or duplicate count %d", count)
		}
		seen[count] = true
	}
	if ttl := store.client.PTTL(ctx, store.prefix+"client").Val(); ttl <= 0 || ttl > time.Second {
		t.Fatalf("counter must expire: %s", ttl)
	}
}

func TestRedisRateLimitRealDoesNotExtendFinalMilliseconds(t *testing.T) {
	store := realRateLimitStore(t)
	ctx := context.Background()
	key := store.prefix + "expiring"
	if err := store.client.Set(ctx, key, "10", 150*time.Millisecond).Err(); err != nil {
		t.Fatal(err)
	}
	count, _, err := store.Incr(ctx, "expiring", time.Minute)
	if err != nil || count != 11 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if ttl := store.client.PTTL(ctx, key).Val(); ttl < 0 || ttl > 150*time.Millisecond {
		t.Fatalf("fixed window was extended: %s", ttl)
	}
}

func BenchmarkRedisRateLimit(b *testing.B) {
	for _, cold := range []bool{false, true} {
		b.Run(fmt.Sprintf("cold=%t", cold), func(b *testing.B) {
			store := realRateLimitStore(b)
			ctx := context.Background()
			if _, _, err := store.Incr(ctx, "warm", time.Hour); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := "warm"
				if cold {
					key = fmt.Sprintf("cold:%d", i)
				}
				if _, _, err := store.Incr(ctx, key, time.Minute); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
