package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache handles caching and fast state storage
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache connection
func NewRedisCache(redisURL string) (*RedisCache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
	}, nil
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// Client returns the underlying Redis client
func (rc *RedisCache) Client() *redis.Client {
	return rc.client
}

// HealthCheck pings Redis to verify connection
func (rc *RedisCache) HealthCheck(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

// Set stores a key-value pair with TTL
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return rc.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key
func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return rc.client.Get(ctx, key).Result()
}

// Delete removes a key
func (rc *RedisCache) Delete(ctx context.Context, keys ...string) error {
	return rc.client.Del(ctx, keys...).Err()
}

