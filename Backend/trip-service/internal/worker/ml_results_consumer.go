package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/services"
)

type MLResultsDeps struct {
	TripRepo             *repositories.TripRepository
	PinRepo              *repositories.PinRepository
	MediaRepo            *repositories.MediaRepository
	TagRepo              *repositories.TagRepository
	PinUploadSessionRepo *repositories.PinUploadSessionRepository
	EventRepo            *repositories.RedisRepository
}

// Возврат error → nak + retry. Возврат nil → ack.
func HandleMLResult(deps MLResultsDeps) repositories.MLResultHandler {
	return func(ctx context.Context, msg repositories.MLResultMessage) error {
		applySimilarGroups(ctx, deps.MediaRepo, msg.SimilarGroups)
		applyNSFW(ctx, deps.MediaRepo, msg.NSFWIDs)
		applyTextModeration(ctx, deps, msg.TextResults)

		switch msg.Flow {
		case repositories.MLFlowCreation, repositories.MLFlowAddMedia:
			return applyPinSuggestionsForTrip(ctx, deps, msg)
		case repositories.MLFlowPinUploadCreation, repositories.MLFlowPinUploadAddition:
			return applyMLResultForPinUpload(ctx, deps, msg)
		case repositories.MLFlowTextModeration:
			return nil
		default:
			slog.WarnContext(ctx, "ml result: unknown flow, ack-drop", "flow", msg.Flow, "trip_id", msg.TripID)
			return nil
		}
	}
}

func applyTextModeration(ctx context.Context, deps MLResultsDeps, results []repositories.MLTextItemResult) {
	if len(results) == 0 {
		return
	}
	for _, r := range results {
		if r.EntityID == "" {
			metrics.MLTextResultSkipped(ctx, "empty_entity_id")
			continue
		}
		val := r.Censored
		switch r.EntityKind {
		case repositories.MLTextEntityTrip:
			if deps.TripRepo == nil {
				metrics.MLTextResultSkipped(ctx, "trip_repo_nil")
				continue
			}
			if err := setTripTextFlag(deps.TripRepo, r.EntityID, r.Field, val); err != nil {
				slog.WarnContext(ctx, "ml text: set trip flag failed",
					"trip_id", r.EntityID, "field", r.Field, "error", err)
				metrics.MLTextResultSkipped(ctx, "set_flag_failed")
				continue
			}
			metrics.MLTextResultApplied(ctx, r.EntityKind, r.Field, val)
		case repositories.MLTextEntityPin:
			if deps.PinRepo == nil {
				metrics.MLTextResultSkipped(ctx, "pin_repo_nil")
				continue
			}
			if err := setPinTextFlag(deps.PinRepo, r.EntityID, r.Field, val); err != nil {
				slog.WarnContext(ctx, "ml text: set pin flag failed",
					"pin_id", r.EntityID, "field", r.Field, "error", err)
				metrics.MLTextResultSkipped(ctx, "set_flag_failed")
				continue
			}
			metrics.MLTextResultApplied(ctx, r.EntityKind, r.Field, val)
		default:
			slog.WarnContext(ctx, "ml text: unknown entity_kind, skipped",
				"entity_kind", r.EntityKind, "entity_id", r.EntityID)
			metrics.MLTextResultSkipped(ctx, "unknown_entity_kind")
		}
	}
}

func setTripTextFlag(repo *repositories.TripRepository, tripID, field string, val bool) error {
	switch field {
	case repositories.MLTextFieldName:
		return repo.SetTextCensored(tripID, &val, nil)
	case repositories.MLTextFieldDescription:
		return repo.SetTextCensored(tripID, nil, &val)
	default:
		return nil
	}
}

func setPinTextFlag(repo *repositories.PinRepository, pinID, field string, val bool) error {
	switch field {
	case repositories.MLTextFieldName:
		return repo.SetTextCensored(pinID, &val, nil)
	case repositories.MLTextFieldDescription:
		return repo.SetTextCensored(pinID, nil, &val)
	default:
		return nil
	}
}

func applySimilarGroups(ctx context.Context, mediaRepo *repositories.MediaRepository, groups [][]string) {
	if mediaRepo == nil || len(groups) == 0 {
		return
	}
	for _, ids := range groups {
		if len(ids) < 2 {
			continue
		}
		groupID := uuid.NewString()
		if err := mediaRepo.SetSimilarGroupID(ids, groupID); err != nil {
			slog.WarnContext(ctx, "ml result: SetSimilarGroupID failed", "error", err)
		}
	}
}

func applyNSFW(ctx context.Context, mediaRepo *repositories.MediaRepository, ids []string) {
	if mediaRepo == nil || len(ids) == 0 {
		return
	}
	if err := mediaRepo.MarkNSFW(ids); err != nil {
		slog.WarnContext(ctx, "ml result: MarkNSFW failed", "error", err)
	}
}

// ML фильтрует suggestions сама — здесь только применяем.
func applyPinSuggestionsForTrip(ctx context.Context, deps MLResultsDeps, msg repositories.MLResultMessage) error {
	if deps.PinRepo == nil || deps.TagRepo == nil {
		return nil
	}
	for _, sug := range msg.PinSuggestions {
		if sug.PinID == "" {
			continue
		}
		pin, err := deps.PinRepo.GetByID(sug.PinID)
		if err != nil {
			slog.WarnContext(ctx, "ml result: pin not found, skipping suggestion", "pin_id", sug.PinID, "error", err)
			continue
		}
		if sug.Category != "" {
			pin.Category = services.ValidatePinCategory(sug.Category)
			pin.Name = pin.Category
		}
		if err := deps.PinRepo.Update(pin); err != nil {
			slog.WarnContext(ctx, "ml result: pin update failed", "pin_id", sug.PinID, "error", err)
			continue
		}
		tags := trimTags(sug.Tags)
		if len(tags) > 0 {
			if err := deps.TagRepo.SetForPin(pin.TripID, pin.ID, tags); err != nil {
				slog.WarnContext(ctx, "ml result: SetForPin failed", "pin_id", sug.PinID, "error", err)
			}
		}
	}
	return nil
}

func applyMLResultForPinUpload(ctx context.Context, deps MLResultsDeps, msg repositories.MLResultMessage) error {
	if deps.PinUploadSessionRepo == nil || msg.SessionID == "" {
		return nil
	}
	session, err := deps.PinUploadSessionRepo.GetByID(ctx, msg.SessionID)
	if err != nil {
		slog.WarnContext(ctx, "ml result: pin_upload session not found", "session_id", msg.SessionID, "error", err)
		return nil
	}
	if session.ProcessingStatus != models.PinUploadProcessingStatusProcessing &&
		session.ProcessingStatus != models.PinUploadProcessingStatusReadyForReview {
		return nil
	}

	var snap pinUploadDraftSnapshotMutator
	if len(session.DraftSnapshot) > 0 {
		if err := json.Unmarshal(session.DraftSnapshot, &snap); err != nil {
			slog.WarnContext(ctx, "ml result: malformed draft_snapshot", "session_id", msg.SessionID, "error", err)
		}
	}

	for _, group := range msg.SimilarGroups {
		if len(group) >= 2 {
			snap.Similar = append(snap.Similar, pinUploadSimilarGroupMutator{MediaIDs: group})
		}
	}
	if len(msg.NSFWIDs) > 0 {
		snap.NSFWMediaIDs = append(snap.NSFWMediaIDs, msg.NSFWIDs...)
	}
	if len(msg.PinSuggestions) > 0 && msg.Flow == repositories.MLFlowPinUploadCreation {
		sug := msg.PinSuggestions[0]
		if snap.Suggested == nil {
			snap.Suggested = &pinSuggestedMutator{}
		}
		if sug.Category != "" {
			snap.Suggested.Category = services.ValidatePinCategory(sug.Category)
			snap.Suggested.Name = snap.Suggested.Category
		}
		if tags := trimTags(sug.Tags); len(tags) > 0 {
			snap.Suggested.Tags = tags
		}
	}

	bytes, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	if err := deps.PinUploadSessionRepo.SetDraftSnapshot(ctx, msg.SessionID, bytes); err != nil {
		slog.WarnContext(ctx, "ml result: SetDraftSnapshot failed", "session_id", msg.SessionID, "error", err)
		return err
	}
	if session.ProcessingStatus == models.PinUploadProcessingStatusProcessing {
		if err := deps.PinUploadSessionRepo.SetProcessingStatus(ctx, msg.SessionID,
			models.PinUploadProcessingStatusProcessing,
			models.PinUploadProcessingStatusReadyForReview); err != nil {
			if !errors.Is(err, repositories.ErrPinUploadSessionWrongState) &&
				!errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
				slog.WarnContext(ctx, "ml result: status transition failed", "session_id", msg.SessionID, "error", err)
			}
		}
	}
	if deps.EventRepo != nil {
		payload := map[string]interface{}{
			"trip_id":           msg.TripID,
			"session_id":        msg.SessionID,
			"processing_status": models.PinUploadProcessingStatusReadyForReview,
		}
		if session.TargetPinID != nil {
			payload["target_pin_id"] = *session.TargetPinID
		}
		_ = deps.EventRepo.PublishTripEventWS(ctx, msg.TripID, repositories.EventPinUploadProcessingCompleted, payload)
	}
	return nil
}

func trimTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	if len(tags) > 10 {
		tags = tags[:10]
	}
	for i, t := range tags {
		if len(t) > 15 {
			tags[i] = t[:15]
		}
	}
	return tags
}

// Зеркало pinUploadDraftSnapshot из services — без него получаем цикл импорта.
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
