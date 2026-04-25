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

func TestMediaRepository_ClusterIDsByLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testinfra.WithTripPostGIS(t)

	sqlDB, err := db.InitDB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	tripRepo := repositories.NewTripRepository(sqlDB)
	mediaRepo := repositories.NewMediaRepository(sqlDB)

	mkTrip := func(t *testing.T) string {
		t.Helper()
		trip := &models.Trip{
			OwnerUserID: uuid.New().String(),
			Name: "Trip",
			Category: "Отпуск",
			Season: "Лето",
			Status: "DRAFT",
			PrivacyLevel: "Private",
		}
		require.NoError(t, tripRepo.Create(trip))
		return trip.ID
	}
	addMedia := func(t *testing.T, tripID string, lat, lon float64) string {
		t.Helper()
		m := &models.Media{
			TripID: tripID,
			S3Key: "k/" + uuid.NewString(),
			MediaType: "image",
			Latitude: &lat,
			Longitude: &lon,
			PrivacyLevel: "Private",
		}
		require.NoError(t, mediaRepo.Create(m))
		return m.ID
	}

	countDistinctClusters := func(m map[string]int) int {
		seen := make(map[int]struct{})
		for _, v := range m {
			seen[v] = struct{}{}
		}
		return len(seen)
	}

	t.Run("same_point_collapses_to_one_cluster", func(t *testing.T) {
		tripID := mkTrip(t)
		for i := 0; i < 3; i++ {
			addMedia(t, tripID, 55.7558, 37.6176)
		}
		clusters, err := mediaRepo.ClusterIDsByLocation(tripID, 75)
		require.NoError(t, err)
		require.Len(t, clusters, 3)
		require.Equal(t, 1, countDistinctClusters(clusters))
	})

	t.Run("distant_cities_form_separate_clusters", func(t *testing.T) {
		tripID := mkTrip(t)
		addMedia(t, tripID, 55.7558, 37.6176) // Moscow
		addMedia(t, tripID, 55.7560, 37.6180) // Moscow (≈30 m)
		addMedia(t, tripID, 40.7128, -74.0060) // New York
		clusters, err := mediaRepo.ClusterIDsByLocation(tripID, 75)
		require.NoError(t, err)
		require.Len(t, clusters, 3)
		require.Equal(t, 2, countDistinctClusters(clusters))
	})

	t.Run("points_just_outside_radius_are_separate", func(t *testing.T) {
		tripID := mkTrip(t)
		// ~120 m apart at latitude 55.7558 (>75 m threshold).
		id1 := addMedia(t, tripID, 55.7558, 37.6176)
		id2 := addMedia(t, tripID, 55.7558, 37.6195)
		clusters, err := mediaRepo.ClusterIDsByLocation(tripID, 75)
		require.NoError(t, err)
		require.Len(t, clusters, 2)
		require.NotEqual(t, clusters[id1], clusters[id2])
	})

	t.Run("empty_trip_returns_empty_map", func(t *testing.T) {
		tripID := mkTrip(t)
		clusters, err := mediaRepo.ClusterIDsByLocation(tripID, 75)
		require.NoError(t, err)
		require.Empty(t, clusters)
	})
}
