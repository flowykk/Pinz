package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/repositories"
)

// pinz:trip:geo_events — обратное направление репликации statistics → trip
// Statistics-service публикует PIN_LOCATIONS_RESOLVED после
// успешного reverse geocoding; trip-service mirror'ит master geo_registry в
// свою read-only реплику и проставляет pins.location_name.
const (
	geoEventsStream    = "pinz:trip:geo_events"
	geoConsumerGroup   = "trip-service-geo-worker"
	geoConsumerName    = "trip-geo-1"
	eventPinLocResolved = "PIN_LOCATIONS_RESOLVED"
)

// RunGeoConsumer запускает consumer-loop pinz:trip:geo_events до отмены ctx.
func RunGeoConsumer(ctx context.Context, redisClient *redis.Client, geoRepo *repositories.GeoRegistryRepository, pinRepo *repositories.PinRepository, eventLog *repositories.GeoEventLogRepository) error {
	if redisClient == nil {
		slog.Warn("geo consumer: redis not configured, disabled")
		<-ctx.Done()
		return nil
	}
	if err := redisClient.XGroupCreateMkStream(ctx, geoEventsStream, geoConsumerGroup, "0").Err(); err != nil && !isBusyGroupErr(err) {
		return err
	}
	slog.Info("geo consumer: started", "stream", geoEventsStream, "group", geoConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    geoConsumerGroup,
			Consumer: geoConsumerName,
			Streams:  []string{geoEventsStream, ">"},
			Count:    50,
			Block:    2 * time.Second,
		}).Result()
		if err != nil && err != redis.Nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.WarnContext(ctx, "geo consumer: XReadGroup error", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				start := time.Now()
				if err := handleGeoMessage(ctx, msg, geoRepo, pinRepo, eventLog); err != nil {
					slog.WarnContext(ctx, "geo consumer: handle failed, skipping ack",
						"msg_id", msg.ID, "error", err)
					metrics.StreamConsumed(ctx, geoEventsStream, geoConsumerGroup, eventPinLocResolved, "error")
					continue
				}
				metrics.StreamConsumed(ctx, geoEventsStream, geoConsumerGroup, eventPinLocResolved, "success")
				metrics.ObserveStreamConsumeDuration(ctx, time.Since(start).Seconds(), geoEventsStream, geoConsumerGroup)
				if err := redisClient.XAck(ctx, stream.Stream, geoConsumerGroup, msg.ID).Err(); err != nil {
					slog.WarnContext(ctx, "geo consumer: XAck failed", "msg_id", msg.ID, "error", err)
				}
			}
		}
	}
}

func handleGeoMessage(ctx context.Context, msg redis.XMessage, geoRepo *repositories.GeoRegistryRepository, pinRepo *repositories.PinRepository, eventLog *repositories.GeoEventLogRepository) error {
	processed, err := eventLog.IsProcessed(ctx, msg.ID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	eventType, _ := msg.Values["event_type"].(string)
	if eventType != eventPinLocResolved {
		// неизвестные события просто помечаем обработанными.
		return eventLog.MarkProcessed(ctx, msg.ID, eventType)
	}
	tripID, _ := msg.Values["trip_id"].(string)
	payload := parseGeoPayload(msg.Values["payload"])
	if err := applyPinLocationsResolved(ctx, tripID, payload, geoRepo, pinRepo); err != nil {
		return err
	}
	return eventLog.MarkProcessed(ctx, msg.ID, eventType)
}

func applyPinLocationsResolved(ctx context.Context, tripID string, payload map[string]any, geoRepo *repositories.GeoRegistryRepository, pinRepo *repositories.PinRepository) error {
	if tripID == "" {
		return nil
	}

	// 1. Mirror master geo_registry строк в локальную реплику.
	geoRows, _ := payload["geo_rows"].([]any)
	for _, item := range geoRows {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		row := repositories.GeoLocation{}
		if v, ok := readInt(m["id"]); ok {
			row.ID = v
		}
		if row.ID == 0 {
			continue
		}
		if v, ok := readInt(m["parent_id"]); ok && v > 0 {
			row.ParentID = &v
		}
		row.Name, _ = m["name"].(string)
		row.Type, _ = m["type"].(string)
		if err := geoRepo.MirrorByID(ctx, row); err != nil {
			slog.WarnContext(ctx, "geo consumer: MirrorByID failed", "trip_id", tripID, "geo_id", row.ID, "error", err)
		}
	}

	// 2. Обновить pins.location_name + trip_locations replica для каждого pin.
	pins, _ := payload["pins"].([]any)
	allLocIDs := make(map[int]struct{})
	for _, item := range pins {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pinID, _ := m["pin_id"].(string)
		locationName, _ := m["location_name"].(string)
		if pinID != "" && locationName != "" {
			if err := pinRepo.UpdateLocationName(pinID, locationName); err != nil {
				slog.WarnContext(ctx, "geo consumer: UpdateLocationName failed", "pin_id", pinID, "error", err)
			}
		}
		if v, ok := readInt(m["country_id"]); ok && v > 0 {
			allLocIDs[v] = struct{}{}
		}
		if v, ok := readInt(m["city_id"]); ok && v > 0 {
			allLocIDs[v] = struct{}{}
		}
	}
	if len(allLocIDs) > 0 {
		ids := make([]int, 0, len(allLocIDs))
		for id := range allLocIDs {
			ids = append(ids, id)
		}
		if err := geoRepo.UpsertTripLocations(ctx, tripID, ids); err != nil {
			slog.WarnContext(ctx, "geo consumer: UpsertTripLocations failed", "trip_id", tripID, "error", err)
		}
	}
	return nil
}

func parseGeoPayload(raw any) map[string]any {
	s, ok := raw.(string)
	if !ok || s == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func readInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	case int:
		return x, true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	}
	return 0, false
}
