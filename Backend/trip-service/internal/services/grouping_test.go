package services

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
)

func TestContentTypeToExt(t *testing.T) {
	cases := map[string]string{
		"image/jpeg": ".jpg",
		"image/jpg": ".jpg",
		"image/png": ".png",
		"image/heic": ".heic",
		"video/mp4": ".mp4",
		"video/quicktime": ".mov",
		"other": ".bin",
		"empty": ".bin",
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			input := name
			if name == "empty" {
				input = ""
			}
			got := contentTypeToExt(input)
			require.Equal(t, want, got, "contentTypeToExt(%q)", input)
		})
	}
}

func ptrFloat(v float64) *float64 { return &v }
func ptrStr(v string) *string { return &v }

func TestClusterMediaToDraftPins_EmptyOrError(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
		mediaRepo.EXPECT().ListByTripID("trip-1").Return(nil, nil)
		require.Nil(t, clusterMediaToDraftPins(mediaRepo, "trip-1"))
	})
	t.Run("list error returns nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
		mediaRepo.EXPECT().ListByTripID("trip-1").Return(nil, errors.New("db down"))
		require.Nil(t, clusterMediaToDraftPins(mediaRepo, "trip-1"))
	})
}

func TestClusterMediaToDraftPins_GeoClustersAndUnassigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	media := []*models.Media{
		{ID: "m1", Latitude: ptrFloat(55.0), Longitude: ptrFloat(37.0), CapturedAt: ptrTime(t0)},
		{ID: "m2", Latitude: ptrFloat(55.0001), Longitude: ptrFloat(37.0001), CapturedAt: ptrTime(t0.Add(time.Minute))},
		{ID: "m3", Latitude: ptrFloat(60.0), Longitude: ptrFloat(40.0), CapturedAt: ptrTime(t0.Add(time.Hour))},
		{ID: "m4", CapturedAt: ptrTime(t0.Add(2 * time.Minute))},  // ближе к кластеру 1 (1 минута)
		{ID: "m5", CapturedAt: ptrTime(t0.Add(48 * time.Hour))},   // далеко по времени → unassigned
		{ID: "m6"},                                                  // без всего → unassigned
	}
	mediaRepo.EXPECT().ListByTripID("trip-1").Return(media, nil)
	mediaRepo.EXPECT().ClusterIDsByLocation("trip-1", float64(ClusterRadiusMeters)).Return(map[string]int{
		"m1": 1, "m2": 1, "m3": 2,
	}, nil)

	got := clusterMediaToDraftPins(mediaRepo, "trip-1")

	require.Len(t, got, 3)
	require.ElementsMatch(t, []string{"m1", "m2", "m4"}, got[0])
	require.ElementsMatch(t, []string{"m3"}, got[1])
	require.ElementsMatch(t, []string{"m5", "m6"}, got[2])
}

func TestClusterMediaToDraftPins_FallbackOnClusteringError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	media := []*models.Media{
		{ID: "m1", Latitude: ptrFloat(55.0), Longitude: ptrFloat(37.0)},
		{ID: "m2", Latitude: ptrFloat(60.0), Longitude: ptrFloat(40.0)},
	}
	mediaRepo.EXPECT().ListByTripID("trip-1").Return(media, nil)
	mediaRepo.EXPECT().ClusterIDsByLocation("trip-1", float64(ClusterRadiusMeters)).Return(nil, errors.New("postgis down"))

	got := clusterMediaToDraftPins(mediaRepo, "trip-1")
	require.Len(t, got, 2)
	require.Contains(t, [][]string{{"m1"}, {"m2"}}, got[0])
}

func TestClusterMediaWithExistingPinsAsSeeds_NoMedia(t *testing.T) {
	ctrl := gomock.NewController(t)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo.EXPECT().ListByTripID("trip-1").Return(nil, nil)
	require.Nil(t, clusterMediaWithExistingPinsAsSeeds(mediaRepo, pinRepo, "trip-1"))
}

func TestClusterMediaWithExistingPinsAsSeeds_NewMediaJoinsExistingPin(t *testing.T) {
	ctrl := gomock.NewController(t)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)

	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	media := []*models.Media{
		{ID: "old1", PinID: ptrStr("pinA"), Latitude: ptrFloat(55.0), Longitude: ptrFloat(37.0), CapturedAt: ptrTime(t0)},
		{ID: "new1", Latitude: ptrFloat(55.0001), Longitude: ptrFloat(37.0001), CapturedAt: ptrTime(t0.Add(time.Minute))},
		{ID: "new2", Latitude: ptrFloat(60.0), Longitude: ptrFloat(40.0)},
		{ID: "new3", CapturedAt: ptrTime(t0.Add(2 * time.Minute))}, // приклеится к pinA по времени
		{ID: "new4"}, // → draft-unassigned
	}
	mediaRepo.EXPECT().ListByTripID("trip-1").Return(media, nil)
	pinRepo.EXPECT().ListByTripID("trip-1").Return([]*models.Pin{{ID: "pinA"}}, nil)
	mediaRepo.EXPECT().ClusterIDsByLocation("trip-1", float64(ClusterRadiusMeters)).Return(map[string]int{
		"old1": 1, "new1": 1, "new2": 2,
	}, nil)

	got := clusterMediaWithExistingPinsAsSeeds(mediaRepo, pinRepo, "trip-1")

	require.Len(t, got, 3)
	require.Equal(t, "existing-pinA", got[0].DraftPinID)
	require.ElementsMatch(t, []string{"old1", "new1", "new3"}, got[0].MediaIDs)
	require.Contains(t, got[1].DraftPinID, "cluster-")
	require.ElementsMatch(t, []string{"new2"}, got[1].MediaIDs)
	require.Equal(t, "draft-unassigned", got[2].DraftPinID)
	require.Equal(t, []string{"new4"}, got[2].MediaIDs)
}

func TestClusterMediaWithExistingPinsAsSeeds_ClusteringErrorFallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)

	media := []*models.Media{
		{ID: "new1", Latitude: ptrFloat(55.0), Longitude: ptrFloat(37.0)},
		{ID: "new2", Latitude: ptrFloat(60.0), Longitude: ptrFloat(40.0)},
	}
	mediaRepo.EXPECT().ListByTripID("trip-1").Return(media, nil)
	pinRepo.EXPECT().ListByTripID("trip-1").Return(nil, nil)
	mediaRepo.EXPECT().ClusterIDsByLocation("trip-1", float64(ClusterRadiusMeters)).Return(nil, errors.New("boom"))

	got := clusterMediaWithExistingPinsAsSeeds(mediaRepo, pinRepo, "trip-1")
	require.Len(t, got, 2)
	for _, g := range got {
		require.Contains(t, g.DraftPinID, "cluster-")
		require.Len(t, g.MediaIDs, 1)
	}
}
