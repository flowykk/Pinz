package worker

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/testinfra"
)

// TestRunPinUploadCleanupOnce_Integration проверяет:
// (1) abandoned-сессия (>72h без активности) закрывается reason=abandoned;
// (2) closed-сессия старше 30d физически удаляется janitor'ом.
func TestRunPinUploadCleanupOnce_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithTripPostGIS(t)

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	tripRepo := repositories.NewTripRepository(sqlDB)
	mediaRepo := repositories.NewMediaRepository(sqlDB)
	sessionRepo := repositories.NewPinUploadSessionRepository(sqlDB)

	ownerID := uuid.New().String()
	trip := &models.Trip{
		OwnerUserID:  ownerID,
		Name:         "Cleanup trip",
		Category:     "Другое",
		Season:       "Лето",
		Status:       "READY",
		PrivacyLevel: "Private",
	}
	require.NoError(t, tripRepo.Create(trip))

	sid, err := sessionRepo.Create(context.Background(), trip.ID, nil, ownerID)
	require.NoError(t, err)

	_, err = sqlDB.Exec(`UPDATE pin_upload_sessions SET last_activity_at = NOW() - INTERVAL '73 hours' WHERE session_id = $1`, sid)
	require.NoError(t, err)

	RunPinUploadCleanupOnce(context.Background(), sessionRepo, mediaRepo, nil)

	session, err := sessionRepo.GetByID(context.Background(), sid)
	require.NoError(t, err)
	require.NotNil(t, session.ClosedAt, "abandoned-сессия должна быть закрыта")

	_, err = sqlDB.Exec(`UPDATE pin_upload_sessions SET closed_at = NOW() - INTERVAL '31 days' WHERE session_id = $1`, sid)
	require.NoError(t, err)

	RunPinUploadCleanupOnce(context.Background(), sessionRepo, mediaRepo, nil)

	_, err = sessionRepo.GetByID(context.Background(), sid)
	require.ErrorIs(t, err, repositories.ErrPinUploadSessionNotFound, "closed > 30 days должен быть физически удалён")
}
