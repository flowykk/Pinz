package worker

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/repositories"
)

const (
	privacyStream = "pinz:trip:privacy:events"
	privacyConsumerGroup = "trip-service-privacy"
	privacyConsumerName = "trip-privacy-1"
)

// ML pipeline переехал в NATS — см. repositories.NATSBroker / worker.HandleMLResult.
func Run(ctx context.Context, redisClient *redis.Client, tripRepo *repositories.TripRepository, pinRepo *repositories.PinRepository, mediaRepo *repositories.MediaRepository, tripPrivacyRepo *repositories.TripPrivacyRepository, pinPrivacyRepo *repositories.PinPrivacyRepository, mediaPrivacyRepo *repositories.MediaPrivacyRepository) error {
	if redisClient == nil {
		slog.Warn("worker: redis not configured, privacy events disabled")
		<-ctx.Done()
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker: context cancelled, stopping")
			return nil
		default:
		}
		if err := processPrivacyEvents(ctx, redisClient, tripRepo, pinRepo, mediaRepo, tripPrivacyRepo, pinPrivacyRepo, mediaPrivacyRepo); err != nil {
			slog.WarnContext(ctx, "worker: processPrivacyEvents failed", "error", err)
		}
	}
}

func isBusyGroupErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}

func processPrivacyEvents(ctx context.Context, client *redis.Client, tripRepo *repositories.TripRepository, pinRepo *repositories.PinRepository, mediaRepo *repositories.MediaRepository, tripPrivacyRepo *repositories.TripPrivacyRepository, pinPrivacyRepo *repositories.PinPrivacyRepository, mediaPrivacyRepo *repositories.MediaPrivacyRepository) error {
	if client == nil {
		return nil
	}
	if err := client.XGroupCreateMkStream(ctx, privacyStream, privacyConsumerGroup, "0").Err(); err != nil && !isBusyGroupErr(err) {
		return err
	}
	streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: privacyConsumerGroup,
		Consumer: privacyConsumerName,
		Streams: []string{privacyStream, ">"},
		Count: 10,
		Block: 1 * time.Second,
	}).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			start := time.Now()
			objType, _ := msg.Values["object_type"].(string)
			objID, _ := msg.Values["object_id"].(string)
			if objID == "" {
				metrics.StreamConsumed(ctx, privacyStream, privacyConsumerGroup, "PRIVACY_CHANGED", "malformed")
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
			metrics.StreamConsumed(ctx, privacyStream, privacyConsumerGroup, "PRIVACY_CHANGED", "success")
			metrics.ObserveStreamConsumeDuration(ctx, time.Since(start).Seconds(), privacyStream, privacyConsumerGroup)
			_ = client.XAck(ctx, stream.Stream, privacyConsumerGroup, msg.ID)
		}
	}
	return nil
}

