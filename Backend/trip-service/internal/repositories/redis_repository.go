package repositories

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

const tripEventsStream = "pinz:trip:events"

// RedisRepository provides Redis client and trip event streaming for Notification Service.
type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// InitRedisClient creates a Redis client. Returns (nil, nil) if REDIS_ADDR and REDIS_URL are both empty (optional Redis).
func InitRedisClient() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		if u := os.Getenv("REDIS_URL"); u != "" {
			opt, err := redis.ParseURL(u)
			if err != nil {
				return nil, err
			}
			client := redis.NewClient(opt)
			if err := client.Ping(context.Background()).Err(); err != nil {
				return nil, fmt.Errorf("redis ping: %w", err)
			}
			instrumentRedis(client)
			return client, nil
		}
		return nil, nil
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	instrumentRedis(client)
	return client, nil
}

func instrumentRedis(client *redis.Client) {
	if err := redisotel.InstrumentTracing(client); err != nil {
		slog.Warn("redis tracing instrumentation failed", "error", err)
	}
	if err := redisotel.InstrumentMetrics(client); err != nil {
		slog.Warn("redis metrics instrumentation failed", "error", err)
	}
}

// PublishTripEvent adds an event to the trip events stream for Notification/Statistics services.
func (r *RedisRepository) PublishTripEvent(ctx context.Context, eventType string, tripID, userID string) error {
	vals := map[string]interface{}{
		"event_type": eventType,
		"trip_id":    tripID,
	}
	if userID != "" {
		vals["user_id"] = userID
	}
	err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: tripEventsStream,
		Values: vals,
	}).Err()
	if err != nil {
		slog.WarnContext(ctx, "PublishTripEvent failed", "event", eventType, "trip_id", tripID, "error", err)
		return err
	}
	return nil
}
