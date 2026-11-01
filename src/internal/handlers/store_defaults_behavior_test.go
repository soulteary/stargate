package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRedisStoresDefaultPrefixPreservesSingleUseTickets(t *testing.T) {
	old := config.SessionStorageRedisKeyPrefix.Value
	t.Cleanup(func() { config.SessionStorageRedisKeyPrefix.Value = old })
	config.SessionStorageRedisKeyPrefix.Value = ""
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	require.Equal(t, "stargate:session:ratelimit:", NewRateLimitStore(client).(*redisRateLimitStore).prefix)
	require.Equal(t, "stargate:session:challenge:context:", NewChallengeContextStore(client).(*redisChallengeContextStore).prefix)
	store := NewSessionExchangeReplayStore(client).(*redisSessionExchangeReplayStore)
	require.Equal(t, "stargate:session:exchange:used:", store.prefix)
	first, err := store.Consume(context.Background(), "ticket-hash", time.Minute)
	require.NoError(t, err)
	require.True(t, first)
	second, err := store.Consume(context.Background(), "ticket-hash", time.Minute)
	require.NoError(t, err)
	require.False(t, second)
}
