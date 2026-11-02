package handlers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type rateLimitReplyHook struct{ reply []int64 }

func (*rateLimitReplyHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (*rateLimitReplyHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *rateLimitReplyHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if err == nil && (cmd.Name() == "eval" || cmd.Name() == "evalsha") {
			switch result := cmd.(type) {
			case *redis.IntSliceCmd:
				result.SetVal(h.reply)
			case *redis.Cmd:
				values := make([]interface{}, len(h.reply))
				for i, value := range h.reply {
					values[i] = value
				}
				result.SetVal(values)
			}
		}
		return err
	}
}

func TestRedisRateLimitRejectsMalformedReplyWithoutRepeatingWrite(t *testing.T) {
	for _, reply := range [][]int64{nil, {1}, {1, 60000, 99}} {
		t.Run(fmt.Sprintf("length=%d", len(reply)), func(t *testing.T) {
			server := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
			t.Cleanup(func() { _ = client.Close() })
			client.AddHook(&rateLimitReplyHook{reply: reply})
			store := NewRateLimitStore(client).(*redisRateLimitStore)
			for expected := 1; expected <= 2; expected++ {
				count, ttl, err := store.Incr(context.Background(), "client", time.Minute)
				require.Error(t, err)
				require.Equal(t, expected, count, "local fallback must still count")
				require.Positive(t, ttl)
				value, err := server.Get(store.prefix + "client")
				require.NoError(t, err)
				require.Equal(t, fmt.Sprint(expected), value, "a malformed reply must not repeat or reset a committed write")
				require.Equal(t, time.Minute, server.TTL(store.prefix+"client"))
			}
		})
	}
}

type rateLimitEvalAdapter struct {
	*redis.Client
	calls  int
	denied error
}

func (a *rateLimitEvalAdapter) EvalSha(context.Context, string, []string, ...interface{}) *redis.Cmd {
	a.calls++
	return redis.NewCmdResult(nil, a.denied)
}

func TestRedisRateLimitPreservesAdapterEvalOverride(t *testing.T) {
	server := miniredis.RunT(t)
	native := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = native.Close() })
	denied := errors.New("adapter denies script execution")
	adapter := &rateLimitEvalAdapter{Client: native, denied: denied}
	store := NewRateLimitStore(adapter)
	count, ttl, err := store.Incr(context.Background(), "client", time.Minute)
	require.ErrorIs(t, err, denied)
	require.Equal(t, 1, count)
	require.Positive(t, ttl)
	require.Equal(t, 1, adapter.calls)
	require.Empty(t, server.Keys(), "promoted Process must not bypass the adapter")
}

func TestRedisRateLimitClusterRoutesScriptByCounterKey(t *testing.T) {
	first, second := miniredis.RunT(t), miniredis.RunT(t)
	client := redis.NewClusterClient(&redis.ClusterOptions{MaxRetries: -1, ClusterSlots: func(context.Context) ([]redis.ClusterSlot, error) {
		return []redis.ClusterSlot{
			{Start: 0, End: 8191, Nodes: []redis.ClusterNode{{Addr: first.Addr()}}},
			{Start: 8192, End: 16383, Nodes: []redis.ClusterNode{{Addr: second.Addr()}}},
		}, nil
	}})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRateLimitStore(client).(*redisRateLimitStore)
	ctx := context.Background()
	destinations := map[*miniredis.Miniredis]bool{}
	for i := 0; i < 16 && len(destinations) < 2; i++ {
		key := fmt.Sprintf("client:%d", i)
		fullKey := store.prefix + key
		require.NoError(t, client.Set(ctx, fullKey, "7", time.Minute).Err())
		var destination *miniredis.Miniredis
		if first.Exists(fullKey) {
			destination = first
		} else {
			destination = second
		}
		destinations[destination] = true
		count, ttl, err := store.Incr(ctx, key, time.Minute)
		require.NoError(t, err)
		require.Equal(t, 8, count)
		require.Equal(t, time.Minute, ttl)
		value, err := destination.Get(fullKey)
		require.NoError(t, err)
		require.Equal(t, "8", value, "script must route by its counter key, not the script digest")
	}
	require.Len(t, destinations, 2, "exercise both routing destinations")
}

func TestRedisRateLimitRingUsesTypedReply(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewRing(&redis.RingOptions{Addrs: map[string]string{"one": server.Addr()}, MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRateLimitStore(client)
	for expected := 1; expected <= 2; expected++ {
		count, ttl, err := store.Incr(context.Background(), "client", time.Minute)
		require.NoError(t, err)
		require.Equal(t, expected, count)
		require.Equal(t, time.Minute, ttl)
	}
}

type normalizedNoScriptHook struct{ calls []string }

func (*normalizedNoScriptHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (*normalizedNoScriptHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *normalizedNoScriptHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "evalsha" || cmd.Name() == "eval" {
			h.calls = append(h.calls, cmd.Name())
			if len(h.calls) == 1 {
				return redis.ErrNoScript
			}
		}
		return next(ctx, cmd)
	}
}
func TestRedisRateLimitReloadsNormalizedNoScript(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	hook := &normalizedNoScriptHook{}
	client.AddHook(hook)
	store := NewRateLimitStore(client)
	count, ttl, err := store.Incr(context.Background(), "client", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, time.Minute, ttl)
	require.Equal(t, []string{"evalsha", "eval"}, hook.calls)
}
