package worker

import (
	"context"
	"log/slog"
	"time"

	"pinz/backend/trip-service/internal/repositories"
)

const (
	GeneratedTripCleanupInterval = 30 * time.Minute
	// Grace prevents race with SaveRecommendation: trip is created before favouriteRepo.Add.
	GeneratedTripCleanupGracePeriod = 5 * time.Minute
	GeneratedTripCleanupBatch       = 100
)

func RunGeneratedTripCleanup(ctx context.Context, tripRepo repositories.TripRepositoryInterface) {
	if tripRepo == nil {
		slog.Warn("generated_trip_cleanup: tripRepo missing, disabled")
		<-ctx.Done()
		return
	}
	slog.Info("generated_trip_cleanup: started",
		"interval", GeneratedTripCleanupInterval,
		"grace", GeneratedTripCleanupGracePeriod,
	)
	ticker := time.NewTicker(GeneratedTripCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("generated_trip_cleanup: context cancelled, stopping")
			return
		case <-ticker.C:
			RunGeneratedTripCleanupOnce(ctx, tripRepo)
		}
	}
}

func RunGeneratedTripCleanupOnce(ctx context.Context, tripRepo repositories.TripRepositoryInterface) {
	ids, err := tripRepo.ListAbandonedGenerated(GeneratedTripCleanupGracePeriod, GeneratedTripCleanupBatch)
	if err != nil {
		slog.WarnContext(ctx, "generated_trip_cleanup: list failed", "err", err)
		return
	}
	for _, id := range ids {
		if err := tripRepo.SetSoftDeleted(id); err != nil {
			slog.WarnContext(ctx, "generated_trip_cleanup: soft-delete failed", "trip_id", id, "err", err)
			continue
		}
		slog.InfoContext(ctx, "generated_trip_cleanup: soft-deleted", "trip_id", id)
	}
}
