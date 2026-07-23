package repositories

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// GeoEventStream — обратное направление репликации (statistics → trip).
// Trip-service consumer'ит этот стрим и mirror'ит geo_registry в свою БД,
// обновляет pins.location_name и trip_locations replica.
const GeoEventStream = "pinz:trip:geo_events"

// GeoEventPublisher описывает точку публикации событий обратного потока.
type GeoEventPublisher interface {
	PublishGeoEvent(ctx context.Context, eventType, tripID string, payload map[string]any) error
}

type RedisGeoPublisher struct {
	client *redis.Client
}

func NewRedisGeoPublisher(client *redis.Client) *RedisGeoPublisher {
	return &RedisGeoPublisher{client: client}
}

func (p *RedisGeoPublisher) PublishGeoEvent(ctx context.Context, eventType, tripID string, payload map[string]any) error {
	if p == nil || p.client == nil {
		return nil
	}
	vals := map[string]any{"event_type": eventType}
	if tripID != "" {
		vals["trip_id"] = tripID
	}
	if len(payload) > 0 {
		if b, err := json.Marshal(payload); err == nil {
			vals["payload"] = string(b)
		}
	}
	if err := p.client.XAdd(ctx, &redis.XAddArgs{Stream: GeoEventStream, Values: vals}).Err(); err != nil {
		slog.WarnContext(ctx, "PublishGeoEvent failed", "event", eventType, "trip_id", tripID, "error", err)
		return err
	}
	return nil
}
