package worker

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/repositories"
)

const (
	mlTasksConsumerGroup = "trip-service-worker"
	mlTasksConsumerName  = "trip-worker-1"
)

// Worker consumes ML/processing tasks from Redis Streams and advances the trip
// creation flow asynchronously (PINZ-99, tripCreationFlow.md).
//
// Responsibilities:
//   - Read tasks from pinz:trip:ml:tasks (created in ApplyGroupsAndProcess)
//   - For each trip_id:
//   - Mark trip status as DRAFT_FINAL_REVIEW
//   - In реальной системе: запуск ML/Reverse Geocoding (пока заглушка)
//   - Уведомить всех участников через Redis Pub/Sub:
//     TRIP_PROCESSING_COMPLETED → API Gateway WebSocket.
func Run(ctx context.Context, redisClient *redis.Client, tripRepo *repositories.TripRepository, participantRepo *repositories.TripParticipantRepository, eventRepo *repositories.RedisRepository) error {
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
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			slog.WarnContext(ctx, "worker: XReadGroup error", "error", err)
			time.Sleep(time.Second)
			continue
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

				if err := processTrip(ctx, tripID, tripRepo, participantRepo, eventRepo); err != nil {
					slog.WarnContext(ctx, "worker: processTrip failed", "trip_id", tripID, "error", err)
				}

				_ = redisClient.XAck(ctx, stream.Stream, mlTasksConsumerGroup, msg.ID)
			}
		}
	}
}

func ensureConsumerGroup(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return nil
	}
	// Create consumer group if it doesn't exist. "0" means deliver all history; для
	// нашего воркера это допустимо (обработка идемпотентна за счёт статуса трипа).
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

func processTrip(ctx context.Context, tripID string, tripRepo *repositories.TripRepository, participantRepo *repositories.TripParticipantRepository, eventRepo *repositories.RedisRepository) error {
	trip, err := tripRepo.GetByID(tripID)
	if err != nil {
		return err
	}

	// Здесь в будущем будет тяжёлая обработка (ML, Reverse Geocoding и т.п.).
	// Сейчас — только смена статуса на DRAFT_FINAL_REVIEW, чтобы разблокировать review-экран.
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
