package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// Compare the native typed reply with the generic compatibility path, using
// the same Lua script, Redis connection settings and already-live counter.
func BenchmarkRedisRateLimitReplyDecoder(b *testing.B) {
	for _, mode := range []string{"native", "generic"} {
		b.Run(mode, func(b *testing.B) {
			store := realRateLimitStore(b)
			if mode == "generic" {
				store.client = struct{ redis.Cmdable }{store.client}
			}
			ctx := context.Background()
			if _, _, err := store.Incr(ctx, "warm", time.Hour); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, _, err := store.Incr(ctx, "warm", time.Minute); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
