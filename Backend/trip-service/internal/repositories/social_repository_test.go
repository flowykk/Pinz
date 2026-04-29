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

func TestSocialRepository_GetReactionsByUserAndTrips(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithTripPostGIS(t)

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	tripRepo := repositories.NewTripRepository(sqlDB)
	socialRepo := repositories.NewSocialRepository(sqlDB)

	mkTrip := func(t *testing.T) string {
		t.Helper()
		trip := &models.Trip{
			OwnerUserID: uuid.New().String(),
			Name: "Trip", Category: "Отпуск", Season: "Лето",
			Status: "DRAFT", PrivacyLevel: "Private",
		}
		require.NoError(t, tripRepo.Create(trip))
		return trip.ID
	}

	user := uuid.New().String()
	other := uuid.New().String()
	tripA := mkTrip(t)
	tripB := mkTrip(t)
	tripC := mkTrip(t)

	_, err = socialRepo.SetReaction(user, tripA, "Like")
	require.NoError(t, err)
	_, err = socialRepo.SetReaction(user, tripB, "Dislike")
	require.NoError(t, err)
	_, err = socialRepo.SetReaction(other, tripC, "Like")
	require.NoError(t, err)

	t.Run("empty_input", func(t *testing.T) {
		out, err := socialRepo.GetReactionsByUserAndTrips(user, nil)
		require.NoError(t, err)
		require.Empty(t, out)
	})

	t.Run("partial_match", func(t *testing.T) {
		out, err := socialRepo.GetReactionsByUserAndTrips(user, []string{tripA, tripB, tripC})
		require.NoError(t, err)
		require.Equal(t, "Like", out[tripA])
		require.Equal(t, "Dislike", out[tripB])
		_, hasC := out[tripC]
		require.False(t, hasC, "tripC принадлежит другому пользователю — в выдачу попасть не должен")
	})

	t.Run("no_match", func(t *testing.T) {
		out, err := socialRepo.GetReactionsByUserAndTrips(user, []string{uuid.NewString()})
		require.NoError(t, err)
		require.Empty(t, out)
	})

	t.Run("invalid_trip_uuid_skipped", func(t *testing.T) {
		out, err := socialRepo.GetReactionsByUserAndTrips(user, []string{"not-a-uuid", tripA})
		require.NoError(t, err)
		require.Equal(t, "Like", out[tripA])
		require.Len(t, out, 1)
	})
}
