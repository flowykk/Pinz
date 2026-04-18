package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

const (
	tripEventsStream    = "pinz:trip:events"
	statsEventsStream   = "pinz:stats:events"
	mlTasksStream       = "pinz:trip:ml:tasks"
	mlResultsStream     = "pinz:trip:ml:results"
	mlContextPrefix     = "pinz:trip:ml:context:"
	privacyEventsStream = "pinz:trip:privacy:events"

	userEventsChannelPrefix = "pinz:user:"
	userEventsChannelSuffix = ":events"

	tripEventsChannelPrefix = "pinz:trip:"
	tripEventsChannelSuffix = ":events"
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

// PublishStatsEvent отправляет событие в stream pinz:stats:events для statistics-service.
// Формат: event_type (string), trip_id (string, опционально), user_ids (JSON []string),
// payload (JSON map[string]any). Если Redis не настроен — no-op.
func (r *RedisRepository) PublishStatsEvent(ctx context.Context, eventType, tripID string, userIDs []string, payload map[string]any) error {
	if r == nil || r.client == nil {
		return nil
	}
	vals := map[string]interface{}{
		"event_type": eventType,
	}
	if tripID != "" {
		vals["trip_id"] = tripID
	}
	if len(userIDs) > 0 {
		if b, err := json.Marshal(userIDs); err == nil {
			vals["user_ids"] = string(b)
		}
	}
	if len(payload) > 0 {
		if b, err := json.Marshal(payload); err == nil {
			vals["payload"] = string(b)
		}
	}
	if err := r.client.XAdd(ctx, &redis.XAddArgs{Stream: statsEventsStream, Values: vals}).Err(); err != nil {
		slog.WarnContext(ctx, "PublishStatsEvent failed", "event", eventType, "trip_id", tripID, "error", err)
		return err
	}
	return nil
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

func (r *RedisRepository) ReadMLResults(ctx context.Context, group, consumer string, count int64, blockMs int64) ([]redis.XStream, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{mlResultsStream, ">"},
		Count:    count,
		Block:    time.Duration(blockMs) * time.Millisecond,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return streams, nil
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

// AddMLTaskWithFlow adds a task with flow marker and optional new pin ids (for add-media).
func (r *RedisRepository) AddMLTaskWithFlow(ctx context.Context, tripID, flow string, newPinIDs []string) error {
	vals := map[string]interface{}{"trip_id": tripID}
	if flow != "" {
		vals["flow"] = flow
	}
	if len(newPinIDs) > 0 {
		if b, err := json.Marshal(newPinIDs); err == nil {
			vals["new_pin_ids"] = string(b)
		}
	}
	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: mlTasksStream,
		Values: vals,
	}).Err()
}

// SetMLContext stores flow-scoped context for later filtering ML results (TTL).
func (r *RedisRepository) SetMLContext(ctx context.Context, tripID, flow string, newPinIDs []string, ttl time.Duration) error {
	if tripID == "" {
		return nil
	}
	vals := map[string]interface{}{}
	if flow != "" {
		vals["flow"] = flow
	}
	if len(newPinIDs) > 0 {
		if b, err := json.Marshal(newPinIDs); err == nil {
			vals["new_pin_ids"] = string(b)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	key := mlContextPrefix + tripID
	if err := r.client.HSet(ctx, key, vals).Err(); err != nil {
		return err
	}
	if ttl > 0 {
		_ = r.client.Expire(ctx, key, ttl).Err()
	}
	return nil
}

// GetMLContext returns flow and new_pin_ids (if present) for the trip.
func (r *RedisRepository) GetMLContext(ctx context.Context, tripID string) (flow string, newPinIDs []string, err error) {
	if tripID == "" {
		return "", nil, nil
	}
	key := mlContextPrefix + tripID
	m, err := r.client.HGetAll(ctx, key).Result()
	if err != nil || len(m) == 0 {
		return "", nil, err
	}
	flow = m["flow"]
	if s := m["new_pin_ids"]; s != "" {
		_ = json.Unmarshal([]byte(s), &newPinIDs)
	}
	return flow, newPinIDs, nil
}

// PublishPrivacyEvent publishes a PRIVACY_CHANGED event for worker aggregation.
func (r *RedisRepository) PublishPrivacyEvent(ctx context.Context, objectType, objectID, tripID, userID, privacyLevel string) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: privacyEventsStream,
		Values: map[string]interface{}{
			"event_type":    "PRIVACY_CHANGED",
			"object_type":   objectType,
			"object_id":     objectID,
			"trip_id":       tripID,
			"user_id":       userID,
			"privacy_level": privacyLevel,
		},
	}).Err()
}

// PublishUserEvent publishes a JSON event to the per-user Pub/Sub channel used by API Gateway
// WebSocket connections. Message format:
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

// PublishTripEventWS fan-outs a WebSocket event to both the per-trip channel
// (consumed by per-resource WS endpoints) and each participant's per-user channel
// (consumed by the global /v1/ws endpoint). Payload is always wrapped into
// {"event","payload"} with trip_id injected so downstream filters work uniformly.
func (r *RedisRepository) PublishTripEventWS(ctx context.Context, tripID string, userIDs []string, eventType string, payload map[string]interface{}) error {
	if r == nil || r.client == nil || tripID == "" {
		return nil
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if _, ok := payload["trip_id"]; !ok {
		payload["trip_id"] = tripID
	}
	data, err := json.Marshal(map[string]interface{}{
		"event":   eventType,
		"payload": payload,
	})
	if err != nil {
		slog.WarnContext(ctx, "PublishTripEventWS marshal failed", "trip_id", tripID, "event", eventType, "error", err)
		return err
	}
	tripChannel := tripEventsChannelPrefix + tripID + tripEventsChannelSuffix
	if err := r.client.Publish(ctx, tripChannel, data).Err(); err != nil {
		slog.WarnContext(ctx, "PublishTripEventWS trip publish failed", "channel", tripChannel, "event", eventType, "error", err)
	}
	for _, uid := range userIDs {
		if uid == "" {
			continue
		}
		userChannel := userEventsChannelPrefix + uid + userEventsChannelSuffix
		if err := r.client.Publish(ctx, userChannel, data).Err(); err != nil {
			slog.WarnContext(ctx, "PublishTripEventWS user publish failed", "channel", userChannel, "event", eventType, "error", err)
		}
	}
	return nil
}
