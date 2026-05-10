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

func TestRunGeneratedTripCleanupOnce_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithTripPostGIS(t)

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	tripRepo := repositories.NewTripRepository(sqlDB)
	favRepo := repositories.NewFavouriteRepository(sqlDB)

	ownerID := uuid.New().String()

	makeGenerated := func(name string) string {
		t.Helper()
		trip := &models.Trip{
			OwnerUserID:  ownerID,
			Name:         name,
			Category:     "custom",
			Season:       "summer",
			Status:       "READY",
			PrivacyLevel: "private",
			IsGenerated:  true,
		}
		require.NoError(t, tripRepo.Create(trip))
		return trip.ID
	}

	abandonedID := makeGenerated("Abandoned")
	favouriteID := makeGenerated("Favourited")
	freshID := makeGenerated("Fresh")

	require.NoError(t, favRepo.Add(ownerID, favouriteID))

	_, err = sqlDB.Exec(`UPDATE trips SET created_at = NOW() - INTERVAL '10 minutes' WHERE id IN ($1, $2)`, abandonedID, favouriteID)
	require.NoError(t, err)

	RunGeneratedTripCleanupOnce(context.Background(), tripRepo)

	abandoned, err := tripRepo.GetByID(abandonedID)
	require.NoError(t, err)
	require.True(t, abandoned.IsSoftDeleted, "трип без favourite должен быть soft-deleted")

	favourited, err := tripRepo.GetByID(favouriteID)
	require.NoError(t, err)
	require.False(t, favourited.IsSoftDeleted, "трип в favourites не должен быть soft-deleted")

	fresh, err := tripRepo.GetByID(freshID)
	require.NoError(t, err)
	require.False(t, fresh.IsSoftDeleted, "свежий трип (< grace) не должен быть soft-deleted")
}
