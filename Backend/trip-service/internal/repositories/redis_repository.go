package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

const (
	tripEventsStream = "pinz:trip:events"
	mlTasksStream    = "pinz:trip:ml:tasks"

	userEventsChannelPrefix = "pinz:user:"
	userEventsChannelSuffix = ":events"
)

// RedisRepository provides Redis client and trip event streaming for Notification/Statistics
// services, as well as Pub/Sub channels for WebSocket notifications via API Gateway.
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

// AddMLTask adds a task to the ML/processing stream (for worker: apply-groups-and-process flow).
func (r *RedisRepository) AddMLTask(ctx context.Context, tripID string) error {
	err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: mlTasksStream,
		Values: map[string]interface{}{"trip_id": tripID},
	}).Err()
	if err != nil {
		slog.WarnContext(ctx, "AddMLTask failed", "trip_id", tripID, "error", err)
		return err
	}
	return nil
}

// PublishUserEvent publishes a JSON event to the per-user Pub/Sub channel used by API Gateway
// WebSocket connections. Message format follows tripCreationFlow.md:
//
//	{
//	  "event": "<event_type>",
//	  "payload": { ... arbitrary JSON ... }
//	}
func (r *RedisRepository) PublishUserEvent(ctx context.Context, userID, eventType string, payload map[string]interface{}) error {
	if r == nil || r.client == nil {
		return nil
	}
	msg := map[string]interface{}{
		"event":   eventType,
		"payload": payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		slog.WarnContext(ctx, "PublishUserEvent marshal failed", "user_id", userID, "event", eventType, "error", err)
		return err
	}
	channel := userEventsChannelPrefix + userID + userEventsChannelSuffix
	if err := r.client.Publish(ctx, channel, data).Err(); err != nil {
		slog.WarnContext(ctx, "PublishUserEvent failed", "channel", channel, "event", eventType, "error", err)
		return err
	}
	return nil
}
