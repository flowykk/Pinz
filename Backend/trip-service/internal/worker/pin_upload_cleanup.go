package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/services"
)

// PinUploadClosedRetention — закрытые сессии старше этого порога удаляются физически.
const PinUploadClosedRetention = 30 * 24 * time.Hour

// RunPinUploadCleanup закрывает pin_upload-сессии без активности дольше SessionAbandonedThreshold
// и физически удаляет закрытые сессии старше PinUploadClosedRetention.
func RunPinUploadCleanup(
	ctx context.Context,
	sessionRepo *repositories.PinUploadSessionRepository,
	mediaRepo *repositories.MediaRepository,
	mediaURLs services.MediaURLResolver,
) {
	if sessionRepo == nil || mediaRepo == nil {
		slog.Warn("pin_upload_cleanup: dependencies missing, disabled")
		<-ctx.Done()
		return
	}
	slog.Info("pin_upload_cleanup: started", "interval", SessionCleanupInterval, "threshold", SessionAbandonedThreshold)
	ticker := time.NewTicker(SessionCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("pin_upload_cleanup: context cancelled, stopping")
			return
		case <-ticker.C:
			RunPinUploadCleanupOnce(ctx, sessionRepo, mediaRepo, mediaURLs)
		}
	}
}

// RunPinUploadCleanupOnce — один проход cleanup-логики (экспортирован для тестов).
func RunPinUploadCleanupOnce(ctx context.Context, sessionRepo *repositories.PinUploadSessionRepository, mediaRepo *repositories.MediaRepository, mediaURLs services.MediaURLResolver) {
	threshold := time.Now().Add(-SessionAbandonedThreshold)
	abandoned, err := sessionRepo.ListAbandoned(ctx, threshold)
	if err != nil {
		slog.WarnContext(ctx, "pin_upload_cleanup: ListAbandoned failed", "err", err)
		return
	}
	for _, a := range abandoned {
		s3Keys, err := mediaRepo.DeleteOrphanByUploadSession(a.SessionID)
		if err != nil {
			slog.WarnContext(ctx, "pin_upload_cleanup: DeleteOrphanByUploadSession failed", "session_id", a.SessionID, "err", err)
		}
		if mediaURLs != nil {
			for _, k := range s3Keys {
				_ = mediaURLs.DeleteObject(ctx, k)
			}
		}
		if err := sessionRepo.Close(ctx, a.SessionID, models.PinUploadSessionCloseReasonAbandoned); err != nil &&
			!errors.Is(err, repositories.ErrPinUploadSessionNotFound) {
			slog.WarnContext(ctx, "pin_upload_cleanup: Close failed", "session_id", a.SessionID, "err", err)
		}
		slog.InfoContext(ctx, "pin_upload_cleanup: cleaned", "session_id", a.SessionID, "trip_id", a.TripID, "deleted_media", len(s3Keys))
	}
	deleted, err := sessionRepo.DeleteClosedOlderThan(ctx, time.Now().Add(-PinUploadClosedRetention))
	if err != nil {
		slog.WarnContext(ctx, "pin_upload_cleanup: DeleteClosedOlderThan failed", "err", err)
		return
	}
	if deleted > 0 {
		slog.InfoContext(ctx, "pin_upload_cleanup: deleted closed sessions", "count", deleted)
	}
}
