package repositories_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/testinfra"
)

func TestFavouriteRepository_FavouritesByUserAndTrips(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithTripPostGIS(t)

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	tripRepo := repositories.NewTripRepository(sqlDB)
	favRepo := repositories.NewFavouriteRepository(sqlDB)

	mkTrip := func(t *testing.T) string {
		t.Helper()
		trip := &models.Trip{
			OwnerUserID: uuid.New().String(),
			Name: "Trip", Category: "vacation", Season: "summer",
			Status: "DRAFT", PrivacyLevel: "private",
		}
		require.NoError(t, tripRepo.Create(trip))
		return trip.ID
	}

	user := uuid.New().String()
	other := uuid.New().String()
	tripA := mkTrip(t)
	tripB := mkTrip(t)
	tripC := mkTrip(t)

	require.NoError(t, favRepo.Add(user, tripA))
	require.NoError(t, favRepo.Add(user, tripB))
	require.NoError(t, favRepo.Add(other, tripC))

	t.Run("empty_input", func(t *testing.T) {
		out, err := favRepo.FavouritesByUserAndTrips(user, nil)
		require.NoError(t, err)
		require.Empty(t, out)
	})

	t.Run("partial_match", func(t *testing.T) {
		out, err := favRepo.FavouritesByUserAndTrips(user, []string{tripA, tripB, tripC})
		require.NoError(t, err)
		_, hasA := out[tripA]
		_, hasB := out[tripB]
		_, hasC := out[tripC]
		require.True(t, hasA)
		require.True(t, hasB)
		require.False(t, hasC, "tripC сохранён другим пользователем — в выдачу попасть не должен")
	})

	t.Run("no_match", func(t *testing.T) {
		out, err := favRepo.FavouritesByUserAndTrips(user, []string{uuid.NewString()})
		require.NoError(t, err)
		require.Empty(t, out)
	})

	t.Run("invalid_trip_uuid_skipped", func(t *testing.T) {
		out, err := favRepo.FavouritesByUserAndTrips(user, []string{"not-a-uuid", tripA})
		require.NoError(t, err)
		_, hasA := out[tripA]
		require.True(t, hasA)
		require.Len(t, out, 1)
	})
}
