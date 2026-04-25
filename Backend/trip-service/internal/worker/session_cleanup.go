package worker

import (
	"context"
	"log/slog"
	"time"

	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/services"
)

// Интервалы для cron session_cleanup .
const (
	// SessionCleanupInterval — раз в 15 минут, достаточно чтобы не блокировать трип
	// более чем на 15 минут после истечения порога.
	SessionCleanupInterval = 15 * time.Minute
	// SessionAbandonedThreshold — сессия без активности дольше 72 часов считается
	// заброшенной и автозакрывается как abandoned.
	SessionAbandonedThreshold = 72 * time.Hour
)

// RunSessionCleanup — периодически ищет заброшенные add-media сессии и закрывает их.
// Процедура на одну сессию:
// 1. Close(session, abandoned).
// 2. Удалить orphan-медиа сессии (pin_id IS NULL, не в existing_media_ids) из БД и S3.
// 3. SetStatus(trip, READY).
// 4. Publish TRIP_STATUS_CHANGED WS для всех participant'ов.
//
// Ошибка на одной сессии не останавливает cycle — логируется и переходим к следующей.
// Завершается при отмене ctx (graceful shutdown).
func RunSessionCleanup(
	ctx context.Context,
	sessionRepo *repositories.AddMediaSessionRepository,
	tripRepo *repositories.TripRepository,
	participantRepo *repositories.TripParticipantRepository,
	mediaRepo *repositories.MediaRepository,
	eventRepo *repositories.RedisRepository,
	mediaURLs services.MediaURLResolver,
) {
	if sessionRepo == nil || tripRepo == nil || mediaRepo == nil {
		slog.Warn("session_cleanup: dependencies missing, disabled")
		<-ctx.Done()
		return
	}
	slog.Info("session_cleanup: started", "interval", SessionCleanupInterval, "threshold", SessionAbandonedThreshold)
	ticker := time.NewTicker(SessionCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("session_cleanup: context cancelled, stopping")
			return
		case <-ticker.C:
			runOnce(ctx, sessionRepo, tripRepo, participantRepo, mediaRepo, eventRepo, mediaURLs)
		}
	}
}

// RunSessionCleanupOnce — один проход cleanup без тикера. Вызывается из
// тестов (integration/e2e) и может быть triggered для ручной очистки.
func RunSessionCleanupOnce(
	ctx context.Context,
	sessionRepo *repositories.AddMediaSessionRepository,
	tripRepo *repositories.TripRepository,
	participantRepo *repositories.TripParticipantRepository,
	mediaRepo *repositories.MediaRepository,
	eventRepo *repositories.RedisRepository,
	mediaURLs services.MediaURLResolver,
) {
	runOnce(ctx, sessionRepo, tripRepo, participantRepo, mediaRepo, eventRepo, mediaURLs)
}

func runOnce(
	ctx context.Context,
	sessionRepo *repositories.AddMediaSessionRepository,
	tripRepo *repositories.TripRepository,
	participantRepo *repositories.TripParticipantRepository,
	mediaRepo *repositories.MediaRepository,
	eventRepo *repositories.RedisRepository,
	mediaURLs services.MediaURLResolver,
) {
	threshold := time.Now().Add(-SessionAbandonedThreshold)
	abandoned, err := sessionRepo.ListAbandoned(ctx, threshold)
	if err != nil {
		slog.WarnContext(ctx, "session_cleanup: ListAbandoned failed", "err", err)
		return
	}
	if len(abandoned) == 0 {
		return
	}
	slog.InfoContext(ctx, "session_cleanup: found abandoned sessions", "count", len(abandoned))
	for _, a := range abandoned {
		cleanupOne(ctx, a, sessionRepo, tripRepo, participantRepo, mediaRepo, eventRepo, mediaURLs)
	}
}

func cleanupOne(
	ctx context.Context,
	a repositories.AbandonedSession,
	sessionRepo *repositories.AddMediaSessionRepository,
	tripRepo *repositories.TripRepository,
	participantRepo *repositories.TripParticipantRepository,
	mediaRepo *repositories.MediaRepository,
	eventRepo *repositories.RedisRepository,
	mediaURLs services.MediaURLResolver,
) {
	existingIDs, _, err := sessionRepo.GetExistingMediaIDs(ctx, a.SessionID)
	if err != nil {
		slog.WarnContext(ctx, "session_cleanup: GetExistingMediaIDs failed", "session_id", a.SessionID, "err", err)
		existingIDs = nil
	}
	s3Keys, err := mediaRepo.DeleteOrphanSessionMedia(a.TripID, existingIDs)
	if err != nil {
		slog.WarnContext(ctx, "session_cleanup: DeleteOrphanSessionMedia failed", "trip_id", a.TripID, "err", err)
	}
	if mediaURLs != nil {
		for _, key := range s3Keys {
			_ = mediaURLs.DeleteObject(ctx, key)
		}
	}
	if _, err := sessionRepo.Close(ctx, a.SessionID, models.AddMediaSessionCloseReasonAbandoned, time.Now()); err != nil {
		slog.WarnContext(ctx, "session_cleanup: Close failed", "session_id", a.SessionID, "err", err)
	}
	if err := tripRepo.SetStatus(a.TripID, models.TripStatusReady); err != nil {
		slog.WarnContext(ctx, "session_cleanup: SetStatus failed", "trip_id", a.TripID, "err", err)
	}
	if eventRepo != nil && participantRepo != nil {
		participants, perr := participantRepo.GetByTripID(a.TripID)
		if perr == nil {
			userIDs := make([]string, 0, len(participants))
			for _, p := range participants {
				userIDs = append(userIDs, p.UserID)
			}
			_ = eventRepo.PublishTripEventWS(ctx, a.TripID, userIDs, repositories.EventTripStatusChanged, map[string]interface{}{
				"trip_id": a.TripID,
				"new_status": models.TripStatusReady,
				"session_id": a.SessionID,
				"reason": "add_media_abandoned",
			})
		}
	}
	slog.InfoContext(ctx, "session_cleanup: cleaned", "session_id", a.SessionID, "trip_id", a.TripID, "deleted_media", len(s3Keys))
}
