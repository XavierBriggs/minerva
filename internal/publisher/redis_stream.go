package publisher

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStreamPublisher publishes events to Redis streams
type RedisStreamPublisher struct {
	client *redis.Client
}

// NewRedisStreamPublisher creates a new Redis stream publisher from existing client
func NewRedisStreamPublisher(client *redis.Client) *RedisStreamPublisher {
	return &RedisStreamPublisher{
		client: client,
	}
}

// RedisPublisher publishes events to Redis streams (legacy name)
type RedisPublisher struct {
	client *redis.Client
}

// NewRedisPublisher creates a new Redis stream publisher
func NewRedisPublisher(redisURL string) (*RedisPublisher, error) {
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

	return &RedisPublisher{
		client: client,
	}, nil
}

// Close closes the Redis connection
func (rp *RedisPublisher) Close() error {
	return rp.client.Close()
}

// PublishLiveGameUpdate publishes a live game update to the stream (for RedisStreamPublisher)
func (rsp *RedisStreamPublisher) PublishLiveGameUpdate(ctx context.Context, gameData interface{}) error {
	streamName := "games.live.basketball_nba"
	
	data, err := json.Marshal(gameData)
	if err != nil {
		return err
	}

	return rsp.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"data":      string(data),
			"timestamp": time.Now().Unix(),
		},
	}).Err()
}

// PublishGameStats publishes final game stats to the stream (for RedisStreamPublisher)
func (rsp *RedisStreamPublisher) PublishGameStats(ctx context.Context, statsData interface{}) error {
	streamName := "games.stats.basketball_nba"
	
	data, err := json.Marshal(statsData)
	if err != nil {
		return err
	}

	return rsp.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"data":      string(data),
			"timestamp": time.Now().Unix(),
		},
	}).Err()
}

// PublishLiveGameUpdate publishes a live game update to the stream (for RedisPublisher)
func (rp *RedisPublisher) PublishLiveGameUpdate(ctx context.Context, gameData interface{}) error {
	streamName := "games.live.basketball_nba"
	
	data, err := json.Marshal(gameData)
	if err != nil {
		return err
	}

	return rp.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"data":      string(data),
			"timestamp": time.Now().Unix(),
		},
	}).Err()
}

// PublishGameStats publishes final game stats to the stream
func (rp *RedisPublisher) PublishGameStats(ctx context.Context, statsData interface{}) error {
	streamName := "games.stats.basketball_nba"
	
	data, err := json.Marshal(statsData)
	if err != nil {
		return err
	}

	return rp.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"data":      string(data),
			"timestamp": time.Now().Unix(),
		},
	}).Err()
}

