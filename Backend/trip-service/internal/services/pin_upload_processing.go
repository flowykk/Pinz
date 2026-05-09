package services

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
)

type PinUploadProcessorDeps struct {
	SessionRepo repositories.PinUploadSessionRepositoryInterface
	MediaRepo   MediaRepositoryForUploadProcessing
	MediaURLs   MediaURLResolver
}

type MediaRepositoryForUploadProcessing interface {
	ListByUploadSession(sessionID string) ([]*models.Media, error)
	ListByPinID(pinID string) ([]*models.Media, error)
	DeleteByIDs(ids []string) error
}

// RunPinUploadProcessing: hash-дедуп, suggested-поля (только creation), pin issues,
// snapshot, CAS PROCESSING→READY_FOR_REVIEW. Возвращает transitioned=true только
// при успешном CAS — caller использует это, чтобы решить, публиковать ли WS.
func RunPinUploadProcessing(ctx context.Context, sessionID string, deps PinUploadProcessorDeps) (bool, error) {
	if deps.SessionRepo == nil || deps.MediaRepo == nil {
		return false, errors.New("pin upload processor: SessionRepo and MediaRepo are required")
	}
	session, err := deps.SessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return false, err
	}

	sessionMedia, err := deps.MediaRepo.ListByUploadSession(sessionID)
	if err != nil {
		return false, err
	}

	// Для addition дедуп идёт против sessionMedia + pin.media.
	var pinMedia []*models.Media
	if session.TargetPinID != nil {
		pinMedia, err = deps.MediaRepo.ListByPinID(*session.TargetPinID)
		if err != nil {
			return false, err
		}
	}
	existingHashes := map[string]struct{}{}
	for _, m := range pinMedia {
		if m.ContentHash != nil {
			existingHashes[*m.ContentHash] = struct{}{}
		}
	}
	seen := map[string]struct{}{}
	var dedupedIDs []string
	var dedupedKeys []string
	for _, m := range sessionMedia {
		if m.ContentHash == nil {
			continue
		}
		if _, dup := existingHashes[*m.ContentHash]; dup {
			dedupedIDs = append(dedupedIDs, m.ID)
			if m.S3Key != "" {
				dedupedKeys = append(dedupedKeys, m.S3Key)
			}
			continue
		}
		if _, dup := seen[*m.ContentHash]; dup {
			dedupedIDs = append(dedupedIDs, m.ID)
			if m.S3Key != "" {
				dedupedKeys = append(dedupedKeys, m.S3Key)
			}
			continue
		}
		seen[*m.ContentHash] = struct{}{}
	}
	if len(dedupedIDs) > 0 {
		if err := deps.MediaRepo.DeleteByIDs(dedupedIDs); err != nil {
			return false, err
		}
		if deps.MediaURLs != nil {
			for _, k := range dedupedKeys {
				_ = deps.MediaURLs.DeleteObject(ctx, k)
			}
		}
	}

	remaining, err := deps.MediaRepo.ListByUploadSession(sessionID)
	if err != nil {
		return false, err
	}

	snap := pinUploadDraftSnapshot{DedupedMediaIDs: dedupedIDs}
	for _, m := range remaining {
		snap.NewMediaIDs = append(snap.NewMediaIDs, m.ID)
	}

	if session.TargetPinID == nil {
		// creation: suggested-поля для нового пина.
		suggested := &pinSuggestedFields{
			Name:     PinCategoryDefault,
			Category: PinCategoryDefault,
			Tags:     []string{},
		}
		for _, m := range remaining {
			if suggested.Latitude == nil && m.Latitude != nil && m.Longitude != nil {
				lat := *m.Latitude
				lon := *m.Longitude
				suggested.Latitude = &lat
				suggested.Longitude = &lon
			}
			if m.CapturedAt != nil {
				ts := m.CapturedAt.Unix()
				if suggested.StartTimeUnix == nil || ts < *suggested.StartTimeUnix {
					v := ts
					suggested.StartTimeUnix = &v
				}
				if suggested.EndTimeUnix == nil || ts > *suggested.EndTimeUnix {
					v := ts
					suggested.EndTimeUnix = &v
				}
			}
		}
		snap.Suggested = suggested
		if suggested.Latitude == nil || suggested.Longitude == nil {
			snap.PinIssues = append(snap.PinIssues, pinIssueMissingCoordinates)
		}
		if suggested.StartTimeUnix == nil {
			snap.PinIssues = append(snap.PinIssues, pinIssueMissingDates)
		}
	} else {
		// addition: pin issues по объединённому набору pin.media + remaining.
		hasDate := false
		hasCoords := false
		for _, m := range pinMedia {
			if m.CapturedAt != nil {
				hasDate = true
			}
			if m.Latitude != nil && m.Longitude != nil {
				hasCoords = true
			}
		}
		for _, m := range remaining {
			if m.CapturedAt != nil {
				hasDate = true
			}
			if m.Latitude != nil && m.Longitude != nil {
				hasCoords = true
			}
		}
		if !hasCoords {
			snap.PinIssues = append(snap.PinIssues, pinIssueMissingCoordinates)
		}
		if !hasDate {
			snap.PinIssues = append(snap.PinIssues, pinIssueMissingDates)
		}
	}

	snapBytes, err := json.Marshal(snap)
	if err != nil {
		return false, err
	}
	if err := deps.SessionRepo.SetDraftSnapshot(ctx, sessionID, snapBytes); err != nil {
		return false, err
	}
	if err := deps.SessionRepo.SetProcessingStatus(ctx, sessionID,
		models.PinUploadProcessingStatusProcessing,
		models.PinUploadProcessingStatusReadyForReview); err != nil {
		if errors.Is(err, repositories.ErrPinUploadSessionWrongState) ||
			errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
			slog.InfoContext(ctx, "RunPinUploadProcessing: session no longer in PROCESSING", "session_id", sessionID)
			return false, nil
		}
		return false, err
	}
	return true, nil
}
