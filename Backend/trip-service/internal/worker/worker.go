package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/repositories"
)

const (
	mlTasksConsumerGroup   = "trip-service-worker"
	mlTasksConsumerName    = "trip-worker-1"
	mlResultsConsumerGroup = "trip-service-ml-results"
	mlResultsConsumerName  = "trip-ml-results-1"
	privacyStream          = "pinz:trip:privacy:events"
	privacyConsumerGroup   = "trip-service-privacy"
	privacyConsumerName    = "trip-privacy-1"
)

// Worker consumes ML/processing tasks from Redis Streams and advances the trip
// creation flow asynchronously.
//
// Responsibilities:
//   - Read tasks from pinz:trip:ml:tasks (created in ApplyGroupsAndProcess)
//   - For each trip_id: mark trip status as DRAFT_FINAL_REVIEW, notify participants via Redis Pub/Sub.
func Run(ctx context.Context, redisClient *redis.Client, tripRepo *repositories.TripRepository, participantRepo *repositories.TripParticipantRepository, geoRepo *repositories.GeoRegistryRepository, mediaRepo *repositories.MediaRepository, tagRepo *repositories.TagRepository, pinRepo *repositories.PinRepository, eventRepo *repositories.RedisRepository, tripPrivacyRepo *repositories.TripPrivacyRepository, pinPrivacyRepo *repositories.PinPrivacyRepository, mediaPrivacyRepo *repositories.MediaPrivacyRepository) error {
	if redisClient == nil || eventRepo == nil {
		slog.Warn("worker: redis not configured, background processing disabled")
		<-ctx.Done()
		return nil
	}

	if err := ensureConsumerGroup(ctx, redisClient); err != nil {
		return err
	}

	slog.Info("worker: started ML task consumer loop", "stream", "pinz:trip:ml:tasks", "group", mlTasksConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker: context cancelled, stopping")
			return nil
		default:
		}

		streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    mlTasksConsumerGroup,
			Consumer: mlTasksConsumerName,
			Streams:  []string{"pinz:trip:ml:tasks", ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()
		if err != nil && err != redis.Nil {
			slog.WarnContext(ctx, "worker: XReadGroup tasks error", "error", err)
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				tripIDRaw, ok := msg.Values["trip_id"]
				if !ok {
					_ = redisClient.XAck(ctx, stream.Stream, mlTasksConsumerGroup, msg.ID)
					continue
				}
				tripID, ok := tripIDRaw.(string)
				if !ok || tripID == "" {
					_ = redisClient.XAck(ctx, stream.Stream, mlTasksConsumerGroup, msg.ID)
					continue
				}

				if err := processTrip(ctx, tripID, tripRepo, participantRepo, geoRepo, eventRepo); err != nil {
					slog.WarnContext(ctx, "worker: processTrip failed", "trip_id", tripID, "error", err)
				}

				_ = redisClient.XAck(ctx, stream.Stream, mlTasksConsumerGroup, msg.ID)
			}
		}

		if err := processMLResults(ctx, redisClient, eventRepo, mediaRepo, tagRepo, pinRepo); err != nil {
			slog.WarnContext(ctx, "worker: processMLResults failed", "error", err)
		}

		if err := processPrivacyEvents(ctx, redisClient, tripRepo, pinRepo, mediaRepo, tripPrivacyRepo, pinPrivacyRepo, mediaPrivacyRepo); err != nil {
			slog.WarnContext(ctx, "worker: processPrivacyEvents failed", "error", err)
		}
	}
}

func ensureConsumerGroup(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return nil
	}
	if err := client.XGroupCreateMkStream(ctx, "pinz:trip:ml:tasks", mlTasksConsumerGroup, "0").Err(); err != nil && !isBusyGroupErr(err) {
		slog.ErrorContext(ctx, "worker: failed to create consumer group", "error", err)
		return err
	}
	return nil
}

func isBusyGroupErr(err error) bool {
	// go-redis возвращает ошибку вида BUSYGROUP Consumer Group name already exists
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}

// processMLResults читает результаты ML (похожие медиа и NSFW) и применяет их к MEDIA.
func processMLResults(ctx context.Context, client *redis.Client, eventRepo *repositories.RedisRepository, mediaRepo *repositories.MediaRepository, tagRepo *repositories.TagRepository, pinRepo *repositories.PinRepository) error {
	if client == nil || mediaRepo == nil {
		return nil
	}
	if err := client.XGroupCreateMkStream(ctx, "pinz:trip:ml:results", mlResultsConsumerGroup, "0").Err(); err != nil && !isBusyGroupErr(err) {
		slog.ErrorContext(ctx, "worker: failed to create ml-results consumer group", "error", err)
		return err
	}

	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    mlResultsConsumerGroup,
		Consumer: mlResultsConsumerName,
		Streams:  []string{"pinz:trip:ml:results", ">"},
		Count:    10,
		Block:    1 * time.Second,
	}).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			tripID := ""
			if raw, ok := msg.Values["trip_id"]; ok {
				if s, ok := raw.(string); ok {
					tripID = s
				}
			}

			// similar_groups: JSON [["media_id1","media_id2"], ...]
			if raw, ok := msg.Values["similar_groups"]; ok {
				if s, ok := raw.(string); ok && s != "" {
					var groups [][]string
					if err := json.Unmarshal([]byte(s), &groups); err == nil {
						for _, ids := range groups {
							if len(ids) < 2 {
								continue
							}
							groupID := uuid.NewString()
							_ = mediaRepo.SetSimilarGroupID(ids, groupID)
						}
					}
				}
			}

			// nsfw_ids: JSON ["media_id3", ...]
			if raw, ok := msg.Values["nsfw_ids"]; ok {
				if s, ok := raw.(string); ok && s != "" {
					var ids []string
					if err := json.Unmarshal([]byte(s), &ids); err == nil && len(ids) > 0 {
						_ = mediaRepo.MarkNSFW(ids)
					}
				}
			}

			// pin_tags: JSON [{"pin_id":"...","category":"...","tags":["t1","t2"]}, ...]
			if raw, ok := msg.Values["pin_tags"]; ok && tagRepo != nil && pinRepo != nil {
				if s, ok := raw.(string); ok && s != "" {
					type pinTagsPayload struct {
						PinID    string   `json:"pin_id"`
						Category string   `json:"category"`
						Tags     []string `json:"tags"`
					}
					var pts []pinTagsPayload
					if err := json.Unmarshal([]byte(s), &pts); err == nil {
						flow := ""
						var allowedNewPins map[string]struct{}
						if eventRepo != nil && tripID != "" {
							if f, newPins, err := eventRepo.GetMLContext(ctx, tripID); err == nil {
								flow = f
								if flow == "add_media" && len(newPins) > 0 {
									allowedNewPins = make(map[string]struct{}, len(newPins))
									for _, id := range newPins {
										allowedNewPins[id] = struct{}{}
									}
								}
							}
						}
						for _, pt := range pts {
							if pt.PinID == "" {
								continue
							}
							if allowedNewPins != nil {
								if _, ok := allowedNewPins[pt.PinID]; !ok {
									// add-media scenario: don't apply auto-tags/category to existing pins
									continue
								}
							}
							pin, err := pinRepo.GetByID(pt.PinID)
							if err != nil {
								continue
							}
							if pt.Category != "" {
								pin.Category = pt.Category
							}
							_ = pinRepo.Update(pin)
							if len(pt.Tags) > 0 {
								_ = tagRepo.SetForPin(pin.TripID, pin.ID, pt.Tags)
							}
						}
					}
				}
			}

			_ = client.XAck(ctx, stream.Stream, mlResultsConsumerGroup, msg.ID)
		}
	}

	return nil
}

func processPrivacyEvents(ctx context.Context, client *redis.Client, tripRepo *repositories.TripRepository, pinRepo *repositories.PinRepository, mediaRepo *repositories.MediaRepository, tripPrivacyRepo *repositories.TripPrivacyRepository, pinPrivacyRepo *repositories.PinPrivacyRepository, mediaPrivacyRepo *repositories.MediaPrivacyRepository) error {
	if client == nil {
		return nil
	}
	if err := client.XGroupCreateMkStream(ctx, privacyStream, privacyConsumerGroup, "0").Err(); err != nil && !isBusyGroupErr(err) {
		return err
	}
	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    privacyConsumerGroup,
		Consumer: privacyConsumerName,
		Streams:  []string{privacyStream, ">"},
		Count:    10,
		Block:    1 * time.Second,
	}).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			objType, _ := msg.Values["object_type"].(string)
			objID, _ := msg.Values["object_id"].(string)
			if objID == "" {
				_ = client.XAck(ctx, stream.Stream, privacyConsumerGroup, msg.ID)
				continue
			}
			switch objType {
			case "trip":
				if tripPrivacyRepo == nil || tripRepo == nil {
					break
				}
				entries, err := tripPrivacyRepo.GetByTripID(ctx, objID)
				if err != nil {
					break
				}
				trip, err := tripRepo.GetByID(objID)
				if err != nil {
					break
				}
				level := repositories.AggregatePrivacyLevel(trip.PrivacyLevel, entries)
				_ = tripRepo.SetPrivacyLevel(objID, level)
			case "pin":
				if pinPrivacyRepo == nil || pinRepo == nil {
					break
				}
				entries, err := pinPrivacyRepo.GetByPinID(ctx, objID)
				if err != nil {
					break
				}
				pin, err := pinRepo.GetByID(objID)
				if err != nil {
					break
				}
				level := repositories.AggregatePrivacyLevel(pin.PrivacyLevel, entries)
				_ = pinRepo.SetPrivacyLevel(objID, level)
			case "media":
				if mediaPrivacyRepo == nil || mediaRepo == nil {
					break
				}
				entries, err := mediaPrivacyRepo.GetByMediaID(ctx, objID)
				if err != nil {
					break
				}
				media, err := mediaRepo.GetByID(objID)
				if err != nil {
					break
				}
				level := repositories.AggregatePrivacyLevel(media.PrivacyLevel, entries)
				_ = mediaRepo.SetPrivacyLevel(objID, level)
			}
			_ = client.XAck(ctx, stream.Stream, privacyConsumerGroup, msg.ID)
		}
	}
	return nil
}

func processTrip(ctx context.Context, tripID string, tripRepo *repositories.TripRepository, participantRepo *repositories.TripParticipantRepository, geoRepo *repositories.GeoRegistryRepository, eventRepo *repositories.RedisRepository) error {
	trip, err := tripRepo.GetByID(tripID)
	if err != nil {
		return err
	}

	if trip.Status != "DRAFT_FINAL_REVIEW" {
		if err := tripRepo.SetStatus(tripID, "DRAFT_FINAL_REVIEW"); err != nil {
			return err
		}
	}

	participants, err := participantRepo.GetByTripID(tripID)
	if err != nil {
		return err
	}

	for _, p := range participants {
		payload := map[string]interface{}{
			"trip_id": tripID,
			"status":  "DRAFT_FINAL_REVIEW",
		}
		_ = eventRepo.PublishUserEvent(ctx, p.UserID, "TRIP_PROCESSING_COMPLETED", payload)
	}

	return nil
}
