package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/statistics-service/internal/metrics"
	"pinz/backend/statistics-service/internal/models"
	"pinz/backend/statistics-service/internal/repositories"
	"pinz/backend/statistics-service/internal/services"
)

const (
	StreamName    = "pinz:stats:events"
	consumerGroup = "statistics-service-worker"
	consumerName  = "stats-worker-1"
)

// Event types — соответствуют publisher'у trip-service.
const (
	EventTripDeleted          = "TRIP_DELETED"
	EventLikeAdded            = "LIKE_ADDED"
	EventLikeRemoved          = "LIKE_REMOVED"
	EventDislikeAdded         = "DISLIKE_ADDED"
	EventDislikeRemoved       = "DISLIKE_REMOVED"
	EventBattleFinished       = "BATTLE_FINISHED"
	EventPinLocationsRequest  = "PIN_LOCATIONS_REQUESTED"
	EventPinLocationsResolved = "PIN_LOCATIONS_RESOLVED"
)

type Deps struct {
	Redis         *redis.Client
	UserStats     repositories.UserStatsRepositoryInterface
	GeoRegistry   repositories.GeoRegistryRepositoryInterface
	TripLocations repositories.TripLocationsRepositoryInterface
	EventLog      repositories.EventLogRepositoryInterface
	Geocoder      services.LocationResolver
	GeoPublisher  repositories.GeoEventPublisher
}

// Run запускает consumer-loop Redis Streams до отмены ctx.
func Run(ctx context.Context, d Deps) error {
	if d.Redis == nil {
		slog.Warn("worker: redis not configured, consumer disabled")
		<-ctx.Done()
		return nil
	}
	if err := ensureGroup(ctx, d.Redis); err != nil {
		return err
	}
	slog.Info("worker: started consumer loop", "stream", StreamName, "group", consumerGroup)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker: context cancelled, stopping")
			return nil
		default:
		}

		streams, err := d.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{StreamName, ">"},
			Count:    50,
			Block:    2 * time.Second,
		}).Result()
		if err != nil && err != redis.Nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.WarnContext(ctx, "worker: XReadGroup error", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				start := time.Now()
				eventType, _ := msg.Values["event_type"].(string)
				if err := handleMessage(ctx, d, msg); err != nil {
					slog.WarnContext(ctx, "worker: handle failed, skipping ack",
						"msg_id", msg.ID, "error", err)
					metrics.StreamConsumed(ctx, StreamName, consumerGroup, eventType, "error")
					continue
				}
				metrics.StreamConsumed(ctx, StreamName, consumerGroup, eventType, "success")
				metrics.ObserveStreamConsumeDuration(ctx, time.Since(start).Seconds(), StreamName, consumerGroup)
				if err := d.Redis.XAck(ctx, stream.Stream, consumerGroup, msg.ID).Err(); err != nil {
					slog.WarnContext(ctx, "worker: XAck failed", "msg_id", msg.ID, "error", err)
				}
			}
		}
	}
}

func ensureGroup(ctx context.Context, client *redis.Client) error {
	if err := client.XGroupCreateMkStream(ctx, StreamName, consumerGroup, "0").Err(); err != nil && !isBusyGroupErr(err) {
		slog.ErrorContext(ctx, "worker: create consumer group failed", "error", err)
		return err
	}
	return nil
}

func isBusyGroupErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}

func handleMessage(ctx context.Context, d Deps, msg redis.XMessage) error {
	processed, err := d.EventLog.IsProcessed(ctx, msg.ID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}

	eventType, _ := msg.Values["event_type"].(string)
	if eventType == "" {
		slog.WarnContext(ctx, "worker: event with empty type, skipping", "msg_id", msg.ID)
		return d.EventLog.MarkProcessed(ctx, msg.ID, "")
	}

	tripID, _ := msg.Values["trip_id"].(string)
	userIDs := parseUserIDs(msg.Values["user_ids"])
	payload := parsePayload(msg.Values["payload"])

	if err := applyEvent(ctx, d, eventType, tripID, userIDs, payload); err != nil {
		return err
	}
	return d.EventLog.MarkProcessed(ctx, msg.ID, eventType)
}

func applyEvent(ctx context.Context, d Deps, eventType, tripID string, userIDs []string, payload map[string]any) error {
	switch eventType {
	case EventPinLocationsRequest:
		return handlePinLocationsRequested(ctx, d, tripID, payload)
	case EventTripDeleted:
		if tripID == "" {
			return nil
		}
		return d.TripLocations.DeleteByTripID(ctx, tripID)
	case EventLikeAdded:
		return applyIncrement(ctx, d.UserStats.IncrementLikes, userIDs, +1)
	case EventLikeRemoved:
		return applyIncrement(ctx, d.UserStats.IncrementLikes, userIDs, -1)
	case EventDislikeAdded:
		return applyIncrement(ctx, d.UserStats.IncrementDislikes, userIDs, +1)
	case EventDislikeRemoved:
		return applyIncrement(ctx, d.UserStats.IncrementDislikes, userIDs, -1)
	case EventBattleFinished:
		return applyIncrement(ctx, d.UserStats.IncrementBattles, userIDs, +1)
	default:
		slog.WarnContext(ctx, "worker: unknown event_type", "event_type", eventType)
	}
	return nil
}

func applyIncrement(ctx context.Context, inc func(context.Context, string, int32) error, userIDs []string, delta int32) error {
	for _, u := range userIDs {
		if u == "" {
			continue
		}
		if err := inc(ctx, u, delta); err != nil {
			return err
		}
	}
	return nil
}

// handlePinLocationsRequested резолвит координаты пинов трипа через BigDataCloud,
// upsert'ит master geo_registry + trip_locations и публикует ответ
// PIN_LOCATIONS_RESOLVED в pinz:trip:geo_events для trip-service.
// Геокодинг — некритичный путь: ошибка на одной точке логируется warning'ом
// и не валит обработку остальных. Если ни одна точка не разрешилась — событие
// считается обработанным, ничего не публикуем (trip-service оставит location_name
// пустым).
func handlePinLocationsRequested(ctx context.Context, d Deps, tripID string, payload map[string]any) error {
	if tripID == "" {
		return nil
	}
	pinsRaw, _ := payload["pins"].([]any)
	if len(pinsRaw) == 0 {
		return nil
	}

	type resolvedPin struct {
		PinID        string
		LocationName string
		CountryID    int32
		CityID       int32
	}
	resolvedPins := make([]resolvedPin, 0, len(pinsRaw))
	geoRows := make(map[int32]*models.GeoLocation)

	for _, item := range pinsRaw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pinID, _ := m["pin_id"].(string)
		lat, latOk := readFloat(m["latitude"])
		lon, lonOk := readFloat(m["longitude"])
		if pinID == "" || !latOk || !lonOk {
			continue
		}

		country, city, name, err := d.Geocoder.ResolveLocation(ctx, lat, lon)
		if err != nil {
			slog.WarnContext(ctx, "worker: geocoding failed", "trip_id", tripID, "pin_id", pinID, "error", err)
			continue
		}
		if name == "" {
			continue
		}
		country = strings.ToLower(strings.TrimSpace(country))
		city = strings.ToLower(strings.TrimSpace(city))
		name = strings.ToLower(strings.TrimSpace(name))

		countryRow, cityRow, err := d.GeoRegistry.EnsureByName(ctx, country, city)
		if err != nil {
			slog.WarnContext(ctx, "worker: EnsureByName failed", "trip_id", tripID, "pin_id", pinID, "error", err)
			continue
		}

		rp := resolvedPin{PinID: pinID, LocationName: name}
		if countryRow != nil {
			geoRows[countryRow.ID] = countryRow
			rp.CountryID = countryRow.ID
			if err := d.TripLocations.Upsert(ctx, tripID, countryRow.ID); err != nil {
				slog.WarnContext(ctx, "worker: TripLocations.Upsert(country) failed", "trip_id", tripID, "error", err)
			}
		}
		if cityRow != nil {
			geoRows[cityRow.ID] = cityRow
			rp.CityID = cityRow.ID
			if err := d.TripLocations.Upsert(ctx, tripID, cityRow.ID); err != nil {
				slog.WarnContext(ctx, "worker: TripLocations.Upsert(city) failed", "trip_id", tripID, "error", err)
			}
		}
		resolvedPins = append(resolvedPins, rp)
	}

	if len(resolvedPins) == 0 {
		return nil
	}

	// Сборка обратного payload.
	geoRowList := make([]map[string]any, 0, len(geoRows))
	for _, row := range geoRows {
		entry := map[string]any{"id": row.ID, "name": row.Name, "type": row.Type}
		if row.ParentID != nil {
			entry["parent_id"] = *row.ParentID
		}
		geoRowList = append(geoRowList, entry)
	}
	pinList := make([]map[string]any, 0, len(resolvedPins))
	for _, rp := range resolvedPins {
		entry := map[string]any{
			"pin_id":        rp.PinID,
			"location_name": rp.LocationName,
		}
		if rp.CountryID != 0 {
			entry["country_id"] = rp.CountryID
		}
		if rp.CityID != 0 {
			entry["city_id"] = rp.CityID
		}
		pinList = append(pinList, entry)
	}
	out := map[string]any{
		"geo_rows": geoRowList,
		"pins":     pinList,
	}
	if d.GeoPublisher != nil {
		if err := d.GeoPublisher.PublishGeoEvent(ctx, EventPinLocationsResolved, tripID, out); err != nil {
			return err
		}
	}
	return nil
}

func readFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

func parseUserIDs(raw any) []string {
	s, ok := raw.(string)
	if !ok || s == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(s), &ids); err != nil {
		return nil
	}
	return ids
}

func parsePayload(raw any) map[string]any {
	s, ok := raw.(string)
	if !ok || s == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}
