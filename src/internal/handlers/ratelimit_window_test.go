package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisRateLimitDoesNotExtendExpiringWindow(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRateLimitStore(client).(*redisRateLimitStore)
	ctx := context.Background()
	if _, _, err := store.Incr(ctx, "client", time.Minute); err != nil {
		t.Fatal(err)
	}
	server.FastForward(time.Minute - 100*time.Millisecond)
	if count, _, err := store.Incr(ctx, "client", time.Minute); err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if ttl := server.TTL(store.prefix + "client"); ttl > 100*time.Millisecond {
		t.Fatalf("request extended the fixed window: TTL=%s", ttl)
	}
	server.FastForward(101 * time.Millisecond)
	if count, _, err := store.Incr(ctx, "client", time.Minute); err != nil || count != 1 {
		t.Fatalf("expired window: count=%d err=%v", count, err)
	}
}

func TestRedisRateLimitRepairsLegacyCounterAndReloadsScript(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRateLimitStore(client).(*redisRateLimitStore)
	ctx := context.Background()
	if err := client.Set(ctx, store.prefix+"legacy", "7", 0).Err(); err != nil {
		t.Fatal(err)
	}
	count, ttl, err := store.Incr(ctx, "legacy", time.Minute)
	if err != nil || count != 8 || ttl != time.Minute {
		t.Fatalf("legacy count=%d ttl=%s err=%v", count, ttl, err)
	}
	if err := client.ScriptFlush(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	count, _, err = store.Incr(ctx, "legacy", time.Minute)
	if err != nil || count != 9 {
		t.Fatalf("reloaded count=%d err=%v", count, err)
	}
	count, ttl, err = store.Incr(ctx, "small", time.Microsecond)
	if err != nil || count != 1 || ttl != time.Millisecond {
		t.Fatalf("sub-millisecond count=%d ttl=%s err=%v", count, ttl, err)
	}
}
