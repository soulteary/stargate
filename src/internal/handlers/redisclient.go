package handlers

import (
	"reflect"

	"github.com/redis/go-redis/v9"
)

// redisClientMissing reports whether client is unusable and the caller should
// fall back to its in-memory implementation.
//
// A plain `client == nil` is not enough. redis.Cmdable is an interface, and the
// callers hold their client as a concrete *redis.Client that is nil whenever
// Redis session storage is disabled. Converting that nil pointer to the
// interface produces a non-nil interface value carrying a nil pointer, so the
// nil check passes, the Redis-backed store is selected, and the first command
// dereferences the nil client. That turned every rate-limited endpoint into a
// panic in the default configuration.
func redisClientMissing(client redis.Cmdable) bool {
	if client == nil {
		return true
	}
	// IsNil is only defined for the kinds that can hold a nil value, and it
	// panics for any other, so the kind is checked first.
	switch value := reflect.ValueOf(client); value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
