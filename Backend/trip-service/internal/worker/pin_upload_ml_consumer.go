package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
)

const (
	pinUploadMLResultsConsumerGroup = "trip-service-pin-upload-ml-results"
	pinUploadMLResultsConsumerName  = "trip-pin-upload-ml-results-1"
)

type PinUploadMLResultsDeps struct {
	SessionRepo *repositories.PinUploadSessionRepository
	MediaRepo   *repositories.MediaRepository
	EventRepo   *repositories.RedisRepository
}

type pinUploadMLResultSuggested struct {
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

func RunPinUploadMLResultsConsumer(ctx context.Context, redisClient *redis.Client, deps PinUploadMLResultsDeps) {
	if redisClient == nil || deps.SessionRepo == nil || deps.MediaRepo == nil {
		slog.Warn("pin_upload_ml_results: dependencies missing, disabled")
		<-ctx.Done()
		return
	}
	if !ensurePinUploadMLResultsGroup(ctx, redisClient) {
		return
	}
	slog.Info("pin_upload_ml_results: started",
		"stream", repositories.PinUploadMLResultsStream,
		"group", pinUploadMLResultsConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			slog.Info("pin_upload_ml_results: context cancelled, stopping")
			return
		default:
		}
		streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    pinUploadMLResultsConsumerGroup,
			Consumer: pinUploadMLResultsConsumerName,
			Streams:  []string{repositories.PinUploadMLResultsStream, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if strings.Contains(err.Error(), "NOGROUP") {
				ensurePinUploadMLResultsGroup(ctx, redisClient)
				continue
			}
			slog.WarnContext(ctx, "pin_upload_ml_results: XReadGroup failed", "error", err)
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				processPinUploadMLResultMessage(ctx, msg, deps)
				_ = redisClient.XAck(ctx, stream.Stream, pinUploadMLResultsConsumerGroup, msg.ID)
			}
		}
	}
}

func ensurePinUploadMLResultsGroup(ctx context.Context, redisClient *redis.Client) bool {
	err := redisClient.XGroupCreateMkStream(ctx, repositories.PinUploadMLResultsStream, pinUploadMLResultsConsumerGroup, "0").Err()
	if err == nil {
		return true
	}
	if strings.Contains(err.Error(), "BUSYGROUP") {
		return true
	}
	slog.ErrorContext(ctx, "pin_upload_ml_results: create group failed", "error", err)
	return false
}

func processPinUploadMLResultMessage(ctx context.Context, msg redis.XMessage, deps PinUploadMLResultsDeps) {
	start := time.Now()
	result := "success"
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "pin_upload_ml_results: panic", "panic", r)
			metrics.Panic(ctx, "pin_upload_ml_results")
			result = "panic"
		}
		metrics.StreamConsumed(ctx, repositories.PinUploadMLResultsStream, pinUploadMLResultsConsumerGroup, "ml_result", result)
		metrics.ObserveStreamConsumeDuration(ctx, time.Since(start).Seconds(), repositories.PinUploadMLResultsStream, pinUploadMLResultsConsumerGroup)
	}()

	tripID, _ := msg.Values["trip_id"].(string)
	sessionID, _ := msg.Values["session_id"].(string)
	if sessionID == "" || tripID == "" {
		slog.WarnContext(ctx, "pin_upload_ml_results: malformed message (missing ids)", "values", msg.Values)
		result = "malformed"
		return
	}

	session, err := deps.SessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		slog.WarnContext(ctx, "pin_upload_ml_results: session not found", "session_id", sessionID, "error", err)
		result = "session_not_found"
		return
	}
	if session.ProcessingStatus != models.PinUploadProcessingStatusProcessing {
		slog.InfoContext(ctx, "pin_upload_ml_results: session no longer in PROCESSING, dropping",
			"session_id", sessionID, "status", session.ProcessingStatus)
		result = "wrong_state"
		return
	}

	var snap pinUploadDraftSnapshotMutator
	if len(session.DraftSnapshot) > 0 {
		if err := json.Unmarshal(session.DraftSnapshot, &snap); err != nil {
			slog.WarnContext(ctx, "pin_upload_ml_results: malformed draft_snapshot", "session_id", sessionID, "error", err)
		}
	}

	if raw, ok := msg.Values["similar_groups"].(string); ok && raw != "" {
		var groups [][]string
		if err := json.Unmarshal([]byte(raw), &groups); err == nil {
			for _, ids := range groups {
				if len(ids) < 2 {
					continue
				}
				groupID := uuid.NewString()
				_ = deps.MediaRepo.SetSimilarGroupID(ids, groupID)
				snap.Similar = append(snap.Similar, pinUploadSimilarGroupMutator{MediaIDs: ids})
			}
		} else {
			slog.WarnContext(ctx, "pin_upload_ml_results: malformed similar_groups", "session_id", sessionID, "error", err)
		}
	}

	if raw, ok := msg.Values["nsfw_ids"].(string); ok && raw != "" {
		var ids []string
		if err := json.Unmarshal([]byte(raw), &ids); err == nil && len(ids) > 0 {
			_ = deps.MediaRepo.MarkNSFW(ids)
			snap.NSFWMediaIDs = append(snap.NSFWMediaIDs, ids...)
		}
	}

	if raw, ok := msg.Values["suggested"].(string); ok && raw != "" && raw != "null" {
		var sg pinUploadMLResultSuggested
		if err := json.Unmarshal([]byte(raw), &sg); err == nil {
			if snap.Suggested == nil {
				snap.Suggested = &pinSuggestedMutator{}
			}
			if sg.Category != "" {
				snap.Suggested.Category = sg.Category
				snap.Suggested.Name = sg.Category
			}
			if len(sg.Tags) > 0 {
				snap.Suggested.Tags = sg.Tags
			}
		}
	}

	snapBytes, err := json.Marshal(snap)
	if err != nil {
		slog.WarnContext(ctx, "pin_upload_ml_results: marshal snapshot", "session_id", sessionID, "error", err)
		result = "marshal_error"
		return
	}
	if err := deps.SessionRepo.SetDraftSnapshot(ctx, sessionID, snapBytes); err != nil {
		slog.WarnContext(ctx, "pin_upload_ml_results: SetDraftSnapshot failed", "session_id", sessionID, "error", err)
		result = "snapshot_save_error"
		return
	}
	if err := deps.SessionRepo.SetProcessingStatus(ctx, sessionID,
		models.PinUploadProcessingStatusProcessing,
		models.PinUploadProcessingStatusReadyForReview); err != nil {
		if errors.Is(err, repositories.ErrPinUploadSessionWrongState) || errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
			result = "wrong_state"
			return
		}
		slog.WarnContext(ctx, "pin_upload_ml_results: SetProcessingStatus failed", "session_id", sessionID, "error", err)
		result = "status_transition_error"
		return
	}
	if deps.EventRepo != nil {
		payload := map[string]interface{}{
			"trip_id":           tripID,
			"session_id":        sessionID,
			"processing_status": models.PinUploadProcessingStatusReadyForReview,
		}
		if session.TargetPinID != nil {
			payload["target_pin_id"] = *session.TargetPinID
		}
		_ = deps.EventRepo.PublishTripEventWS(ctx, tripID, repositories.EventPinUploadProcessingCompleted, payload)
	}
	metrics.PinUploadSession(ctx, "ml_result", "applied")
}

// Зеркала структур из services/pin_upload.go — separate types в worker'е
// нужны, чтобы избежать циклического импорта services <-> worker.
type pinUploadDraftSnapshotMutator struct {
	Suggested       *pinSuggestedMutator           `json:"suggested,omitempty"`
	NewMediaIDs     []string                       `json:"new_media_ids"`
	NSFWMediaIDs    []string                       `json:"nsfw_media_ids"`
	DedupedMediaIDs []string                       `json:"deduped_media_ids"`
	PinIssues       []string                       `json:"pin_issues"`
	Similar         []pinUploadSimilarGroupMutator `json:"similar"`
}

type pinSuggestedMutator struct {
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	StartTimeUnix *int64   `json:"start_time_unix,omitempty"`
	EndTimeUnix   *int64   `json:"end_time_unix,omitempty"`
}

type pinUploadSimilarGroupMutator struct {
	MediaIDs []string `json:"media_ids"`
}
