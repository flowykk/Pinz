package worker

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/services"
)

const (
	pinUploadTasksStream = "pinz:trip:pin_upload:tasks"
	pinUploadConsumerGroup = "trip-service-pin-upload"
)

// PinUploadConsumerCount — число параллельных consumer-горутин в одной group.
const PinUploadConsumerCount = 4

type PinUploadConsumerDeps struct {
	SessionRepo *repositories.PinUploadSessionRepository
	MediaRepo   *repositories.MediaRepository
	EventRepo   *repositories.RedisRepository
	MediaURLs   services.MediaURLResolver
}

// RunPinUploadConsumer — один consumer-loop. Стартовать N штук c разными consumerName.
func RunPinUploadConsumer(ctx context.Context, redisClient *redis.Client, consumerName string, deps PinUploadConsumerDeps) {
	if redisClient == nil || deps.SessionRepo == nil || deps.MediaRepo == nil {
		slog.Warn("pin_upload_consumer: dependencies missing, disabled", "consumer", consumerName)
		<-ctx.Done()
		return
	}
	if !ensurePinUploadGroup(ctx, redisClient, consumerName) {
		return
	}
	slog.Info("pin_upload_consumer: started", "stream", pinUploadTasksStream, "group", pinUploadConsumerGroup, "consumer", consumerName)
	for {
		select {
		case <-ctx.Done():
			slog.Info("pin_upload_consumer: context cancelled, stopping", "consumer", consumerName)
			return
		default:
		}
		streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    pinUploadConsumerGroup,
			Consumer: consumerName,
			Streams:  []string{pinUploadTasksStream, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			slog.WarnContext(ctx, "pin_upload_consumer: XReadGroup failed", "consumer", consumerName, "error", err)
			if isNoGroupErr(err) && !ensurePinUploadGroup(ctx, redisClient, consumerName) {
				return
			}
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				processPinUploadMessage(ctx, msg, deps)
				_ = redisClient.XAck(ctx, stream.Stream, pinUploadConsumerGroup, msg.ID)
			}
		}
	}
}

func ensurePinUploadGroup(ctx context.Context, redisClient *redis.Client, consumerName string) bool {
	backoff := 500 * time.Millisecond
	const maxBackoff = 30 * time.Second
	for {
		err := redisClient.XGroupCreateMkStream(ctx, pinUploadTasksStream, pinUploadConsumerGroup, "0").Err()
		if err == nil || isBusyGroupErr(err) {
			return true
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		slog.WarnContext(ctx, "pin_upload_consumer: create group failed, retrying",
			"consumer", consumerName, "error", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return false
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func isNoGroupErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "NOGROUP")
}

func processPinUploadMessage(ctx context.Context, msg redis.XMessage, deps PinUploadConsumerDeps) {
	tripID, _ := msg.Values["trip_id"].(string)
	sessionID, _ := msg.Values["session_id"].(string)
	initiator, _ := msg.Values["initiator_user_id"].(string)
	targetPinID, _ := msg.Values["target_pin_id"].(string)
	if sessionID == "" || tripID == "" {
		slog.WarnContext(ctx, "pin_upload_consumer: malformed message", "values", msg.Values)
		metrics.StreamConsumed(ctx, pinUploadTasksStream, pinUploadConsumerGroup, "pin_upload_task", "malformed")
		return
	}
	start := time.Now()
	result := "success"
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "pin_upload_consumer: panic in handler",
				"session_id", sessionID, "trip_id", tripID, "panic", r)
			metrics.Panic(ctx, "pin_upload_consumer")
			result = "panic"
		}
		metrics.StreamConsumed(ctx, pinUploadTasksStream, pinUploadConsumerGroup, "pin_upload_task", result)
		metrics.ObserveStreamConsumeDuration(ctx, time.Since(start).Seconds(), pinUploadTasksStream, pinUploadConsumerGroup)
		metrics.ObservePinUploadDuration(ctx, time.Since(start).Seconds(), result)
	}()
	slog.InfoContext(ctx, "pin_upload_consumer: got task",
		"session_id", sessionID, "trip_id", tripID, "message_id", msg.ID)
	transitioned, err := services.RunPinUploadProcessing(ctx, sessionID, services.PinUploadProcessorDeps{
		SessionRepo: deps.SessionRepo,
		MediaRepo:   deps.MediaRepo,
		MediaURLs:   deps.MediaURLs,
	})
	if err != nil {
		slog.WarnContext(ctx, "pin_upload_consumer: processing failed",
			"session_id", sessionID, "trip_id", tripID, "error", err)
		// fallback: всё равно довести до READY_FOR_REVIEW, чтобы клиент мог finalize.
		if ferr := deps.SessionRepo.SetProcessingStatus(ctx, sessionID,
			models.PinUploadProcessingStatusProcessing,
			models.PinUploadProcessingStatusReadyForReview); ferr != nil {
			if !errors.Is(ferr, repositories.ErrPinUploadSessionWrongState) &&
				!errors.Is(ferr, repositories.ErrPinUploadSessionNotFound) {
				slog.WarnContext(ctx, "pin_upload_consumer: fallback transition failed",
					"session_id", sessionID, "error", ferr)
			}
			result = "fallback_failed"
		} else {
			transitioned = true
			result = "fallback"
			metrics.PinUploadSession(ctx, "process", "fallback")
		}
	}
	if !transitioned {
		if result == "success" {
			result = "no_transition"
		}
		return
	}
	if result == "success" {
		metrics.PinUploadSession(ctx, "process", "success")
	}
	publishPinUploadMLTaskDryRun(ctx, deps, tripID, sessionID, targetPinID)
	if deps.EventRepo != nil {
		payload := map[string]interface{}{
			"trip_id":           tripID,
			"session_id":        sessionID,
			"initiator_user_id": initiator,
			"processing_status": models.PinUploadProcessingStatusReadyForReview,
		}
		if targetPinID != "" {
			payload["target_pin_id"] = targetPinID
		}
		_ = deps.EventRepo.PublishTripEventWS(ctx, tripID,
			repositories.EventPinUploadProcessingCompleted, payload)
	}
}

// dry-run параллельно со stub'ом для валидации ML-контракта. Ошибки swallowed.
func publishPinUploadMLTaskDryRun(ctx context.Context, deps PinUploadConsumerDeps, tripID, sessionID, targetPinID string) {
	if deps.EventRepo == nil || deps.MediaURLs == nil || deps.MediaRepo == nil {
		return
	}
	sessionMedia, err := deps.MediaRepo.ListByUploadSession(sessionID)
	if err != nil {
		slog.WarnContext(ctx, "ml dry-run: ListByUploadSession failed",
			"session_id", sessionID, "trip_id", tripID, "error", err)
		return
	}
	if len(sessionMedia) == 0 {
		return
	}
	var pinMedia []*models.Media
	if targetPinID != "" {
		pinMedia, err = deps.MediaRepo.ListByPinID(targetPinID)
		if err != nil {
			slog.WarnContext(ctx, "ml dry-run: ListByPinID failed",
				"session_id", sessionID, "trip_id", tripID, "target_pin_id", targetPinID, "error", err)
			return
		}
	}
	flowLabel := services.MLFlowPinUploadCreate
	if targetPinID != "" {
		flowLabel = services.MLFlowPinUploadAddTo
	}
	newJSON, existingJSON, expiresAtUnix, _, err := services.BuildPinUploadMLPayload(ctx, flowLabel, sessionMedia, pinMedia, deps.MediaURLs)
	if err != nil {
		slog.WarnContext(ctx, "ml dry-run: BuildPinUploadMLPayload failed",
			"session_id", sessionID, "trip_id", tripID, "error", err)
		return
	}
	if err := deps.EventRepo.AddPinUploadMLTask(ctx, tripID, sessionID, targetPinID, newJSON, existingJSON, expiresAtUnix); err != nil {
		slog.WarnContext(ctx, "ml dry-run: AddPinUploadMLTask failed",
			"session_id", sessionID, "trip_id", tripID, "error", err)
	}
}
