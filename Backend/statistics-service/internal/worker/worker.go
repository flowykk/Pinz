package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/statistics-service/internal/models"
	"pinz/backend/statistics-service/internal/repositories"
)

const (
	StreamName    = "pinz:stats:events"
	consumerGroup = "statistics-service-worker"
	consumerName  = "stats-worker-1"
)

// Event types — соответствуют publisher'у trip-service (RedisRepository.PublishStatsEvent).
const (
	EventTripDeleted        = "TRIP_DELETED"         // cleanup trip_locations_mirror
	EventTripLocationsAdded = "TRIP_LOCATIONS_ADDED" // upsert mirror
	EventLikeAdded          = "LIKE_ADDED"
	EventLikeRemoved        = "LIKE_REMOVED"
	EventDislikeAdded       = "DISLIKE_ADDED"
	EventDislikeRemoved     = "DISLIKE_REMOVED"
	EventBattleFinished     = "BATTLE_FINISHED"
)

type Deps struct {
	Redis        *redis.Client
	UserStats    repositories.UserStatsRepositoryInterface
	GeoRegistry repositories.GeoRegistryRepositoryInterface
	TripLocations repositories.TripLocationsRepositoryInterface
	EventLog     repositories.EventLogRepositoryInterface
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
				if err := handleMessage(ctx, d, msg); err != nil {
					slog.WarnContext(ctx, "worker: handle failed, skipping ack",
						"msg_id", msg.ID, "error", err)
					continue
				}
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
	case EventTripLocationsAdded:
		return applyTripLocations(ctx, d, tripID, payload)
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

func applyTripLocations(ctx context.Context, d Deps, tripID string, payload map[string]any) error {
	if tripID == "" {
		return nil
	}
	raw, ok := payload["locations"]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		loc := parseLocation(m)
		if loc == nil || loc.ID == 0 {
			continue
		}
		if err := d.GeoRegistry.Upsert(ctx, loc); err != nil {
			return err
		}
		if err := d.TripLocations.Upsert(ctx, tripID, loc.ID); err != nil {
			return err
		}
	}
	return nil
}

func parseLocation(m map[string]any) *models.GeoLocation {
	loc := &models.GeoLocation{}
	if v, ok := m["id"].(float64); ok {
		loc.ID = int32(v)
	}
	if v, ok := m["parent_id"].(float64); ok && v > 0 {
		p := int32(v)
		loc.ParentID = &p
	}
	if v, ok := m["name"].(string); ok {
		loc.Name = v
	}
	if v, ok := m["type"].(string); ok {
		loc.Type = v
	}
	return loc
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
