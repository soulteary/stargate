package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	rediskitclient "github.com/soulteary/redis-kit/client"
)

// RedisStorage implements fiber.Storage interface for Redis-based session storage
type RedisStorage struct {
	client    *redis.Client
	keyPrefix string
}

// NewRedisStorage creates a new Redis storage for Fiber sessions
// This implements the fiber.Storage interface using redis-kit client
func NewRedisStorage(redisClient *redis.Client, keyPrefix string) fiber.Storage {
	if keyPrefix == "" {
		keyPrefix = "stargate:session:"
	} else if len(keyPrefix) > 0 && keyPrefix[len(keyPrefix)-1] != ':' {
		keyPrefix += ":"
	}

	logrus.Info("Redis session storage initialized with prefix: ", keyPrefix)
	return &RedisStorage{
		client:    redisClient,
		keyPrefix: keyPrefix,
	}
}

// buildKey constructs the full key with prefix
func (s *RedisStorage) buildKey(key string) string {
	return s.keyPrefix + key
}

// Get gets the value for the given key.
// `nil, nil` is returned when the key does not exist
func (s *RedisStorage) Get(key string) ([]byte, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	fullKey := s.buildKey(key)
	ctx := context.Background()

	data, err := s.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return nil, nil // Key does not exist, return nil, nil as per interface
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from redis: %w", err)
	}

	return data, nil
}

// Set stores the given value for the given key along with an expiration value.
// 0 means no expiration. Empty key or value will be ignored without an error.
func (s *RedisStorage) Set(key string, val []byte, exp time.Duration) error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	if key == "" || len(val) == 0 {
		return nil // Ignore empty key or value as per interface
	}

	fullKey := s.buildKey(key)
	ctx := context.Background()

	if exp > 0 {
		err := s.client.Set(ctx, fullKey, val, exp).Err()
		if err != nil {
			return fmt.Errorf("failed to set in redis: %w", err)
		}
	} else {
		err := s.client.Set(ctx, fullKey, val, 0).Err()
		if err != nil {
			return fmt.Errorf("failed to set in redis: %w", err)
		}
	}

	return nil
}

// Delete deletes the value for the given key.
// It returns no error if the storage does not contain the key.
func (s *RedisStorage) Delete(key string) error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	fullKey := s.buildKey(key)
	ctx := context.Background()

	err := s.client.Del(ctx, fullKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete from redis: %w", err)
	}

	return nil
}

// Reset resets the storage and delete all keys.
func (s *RedisStorage) Reset() error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	ctx := context.Background()

	// Get all keys matching the prefix
	pattern := s.keyPrefix + "*"
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	// Delete all keys
	if len(keys) > 0 {
		err := s.client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}

	return nil
}

// Close closes the storage and will stop any running garbage collectors and open connections.
func (s *RedisStorage) Close() error {
	if s.client == nil {
		return nil
	}

	err := rediskitclient.Close(s.client)
	if err != nil {
		return fmt.Errorf("failed to close redis client: %w", err)
	}

	return nil
}

// NewRedisClientFromConfig creates a Redis client using redis-kit with configuration
func NewRedisClientFromConfig(addr, password string, db int) (*redis.Client, error) {
	cfg := rediskitclient.DefaultConfig().
		WithAddr(addr).
		WithPassword(password).
		WithDB(db)

	client, err := rediskitclient.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rediskitclient.Ping(ctx, client); err != nil {
		_ = rediskitclient.Close(client)
		return nil, err
	}

	logrus.Info("Redis client connected successfully: ", addr)
	return client, nil
}
