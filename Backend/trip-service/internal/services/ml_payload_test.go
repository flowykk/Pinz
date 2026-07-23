package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
)

func mediaFixture(id, s3Key string, capturedAtUnix int64, lat, lon float64) *models.Media {
	captured := time.Unix(capturedAtUnix, 0)
	return &models.Media{
		ID:         id,
		S3Key:      s3Key,
		MediaType:  "image",
		Latitude:   &lat,
		Longitude:  &lon,
		CapturedAt: &captured,
	}
}

func TestBuildMLTaskMessageForTrip_Creation_AllPinsNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	pin1 := &models.Pin{ID: "pin-1", TripID: "trip-1"}
	pin2 := &models.Pin{ID: "pin-2", TripID: "trip-1"}
	pinRepo.EXPECT().ListByTripID("trip-1").Return([]*models.Pin{pin1, pin2}, nil)
	mediaRepo.EXPECT().ListByPinID("pin-1").Return([]*models.Media{
		mediaFixture("m1", "k1", 1000, 1, 2),
	}, nil)
	mediaRepo.EXPECT().ListByPinID("pin-2").Return([]*models.Media{
		mediaFixture("m2", "k2", 2000, 3, 4),
	}, nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k1", mlPresignTTL).Return("https://signed/k1", nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k2", mlPresignTTL).Return("https://signed/k2", nil)

	msg, count, err := BuildMLTaskMessageForTrip(context.Background(),
		"trip-1", MLFlowCreation, "", nil, nil, pinRepo, mediaRepo, urls)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Greater(t, msg.ExpiresAtUnix, time.Now().Unix())
	require.Equal(t, MLFlowCreation, msg.Flow)
	require.Len(t, msg.Pins, 2)
	require.True(t, msg.Pins[0].IsNew)
	require.True(t, msg.Pins[1].IsNew)
	// is_new=true и на уровне media для new pin'ов.
	require.True(t, msg.Pins[0].Media[0].IsNew)
	require.Equal(t, "https://signed/k1", msg.Pins[0].Media[0].GetURL)
	require.Equal(t, int64(1000), *msg.Pins[0].Media[0].CapturedAtUnix)
}

func TestBuildMLTaskMessageForTrip_AddMedia_ExistingPinAllMediaIsNewFlagged(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	newPin := &models.Pin{ID: "pin-new", TripID: "trip-1"}
	existingPin := &models.Pin{ID: "pin-existing", TripID: "trip-1"}
	pinRepo.EXPECT().ListByTripID("trip-1").Return([]*models.Pin{newPin, existingPin}, nil)

	// New pin: всё медиа уезжает как is_new=true.
	mediaRepo.EXPECT().ListByPinID("pin-new").Return([]*models.Media{
		mediaFixture("m-new-1", "k-new-1", 1000, 1, 2),
	}, nil)
	// Existing pin: все медиа уезжают, но только m-added-1 помечается is_new=true.
	mediaRepo.EXPECT().ListByPinID("pin-existing").Return([]*models.Media{
		mediaFixture("m-orig-1", "k-orig-1", 500, 5, 6),
		mediaFixture("m-added-1", "k-added-1", 2000, 7, 8),
	}, nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k-new-1", mlPresignTTL).Return("https://signed/new", nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k-orig-1", mlPresignTTL).Return("https://signed/orig", nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k-added-1", mlPresignTTL).Return("https://signed/added", nil)

	msg, count, err := BuildMLTaskMessageForTrip(context.Background(),
		"trip-1", MLFlowAddMedia, "sess-1",
		[]string{"pin-new"},
		[]string{"m-added-1"},
		pinRepo, mediaRepo, urls)
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Equal(t, "sess-1", msg.SessionID)
	require.Len(t, msg.Pins, 2)

	byID := map[string]repositories.MLPinPayload{msg.Pins[0].PinID: msg.Pins[0], msg.Pins[1].PinID: msg.Pins[1]}
	require.True(t, byID["pin-new"].IsNew)
	require.True(t, byID["pin-new"].Media[0].IsNew)

	existing := byID["pin-existing"]
	require.False(t, existing.IsNew)
	require.Len(t, existing.Media, 2)
	mediaByID := map[string]repositories.MLMediaPayload{existing.Media[0].MediaID: existing.Media[0], existing.Media[1].MediaID: existing.Media[1]}
	require.False(t, mediaByID["m-orig-1"].IsNew)
	require.True(t, mediaByID["m-added-1"].IsNew)
}

func TestBuildMLTaskMessageForTrip_AddMedia_PinWithoutPending_StillIncluded(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	existingPin := &models.Pin{ID: "pin-untouched", TripID: "trip-1"}
	pinRepo.EXPECT().ListByTripID("trip-1").Return([]*models.Pin{existingPin}, nil)
	mediaRepo.EXPECT().ListByPinID("pin-untouched").Return([]*models.Media{
		mediaFixture("m1", "k1", 0, 0, 0),
	}, nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k1", mlPresignTTL).Return("https://signed/k1", nil)

	msg, count, err := BuildMLTaskMessageForTrip(context.Background(),
		"trip-1", MLFlowAddMedia, "", nil, nil, pinRepo, mediaRepo, urls)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Len(t, msg.Pins, 1)
	require.False(t, msg.Pins[0].IsNew)
	require.False(t, msg.Pins[0].Media[0].IsNew)
}

func TestBuildMLTaskMessageForTrip_NoCoordsNoCaptured_OmitFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	pin := &models.Pin{ID: "pin-1", TripID: "trip-1"}
	pinRepo.EXPECT().ListByTripID("trip-1").Return([]*models.Pin{pin}, nil)
	mediaRepo.EXPECT().ListByPinID("pin-1").Return([]*models.Media{
		{ID: "m1", S3Key: "k1", MediaType: "image"},
	}, nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k1", mlPresignTTL).Return("https://signed/k1", nil)

	msg, _, err := BuildMLTaskMessageForTrip(context.Background(),
		"trip-1", MLFlowCreation, "", nil, nil, pinRepo, mediaRepo, urls)
	require.NoError(t, err)
	require.Nil(t, msg.Pins[0].Media[0].Latitude)
	require.Nil(t, msg.Pins[0].Media[0].CapturedAtUnix)
}

func TestBuildMLTaskMessageForTrip_UnknownFlow_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	_, _, err := BuildMLTaskMessageForTrip(context.Background(),
		"trip-1", "bogus", "", nil, nil, pinRepo, mediaRepo, urls)
	require.Error(t, err)
}

func TestBuildMLTaskMessageForTrip_PresignError_Propagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	pin := &models.Pin{ID: "pin-1"}
	pinRepo.EXPECT().ListByTripID("trip-1").Return([]*models.Pin{pin}, nil)
	mediaRepo.EXPECT().ListByPinID("pin-1").Return([]*models.Media{
		mediaFixture("m1", "k1", 0, 0, 0),
	}, nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k1", mlPresignTTL).Return("", errors.New("boom"))

	_, _, err := BuildMLTaskMessageForTrip(context.Background(),
		"trip-1", MLFlowCreation, "", nil, nil, pinRepo, mediaRepo, urls)
	require.Error(t, err)
}

func TestBuildMLTaskMessageForPinUpload_Creation_OnlyNewMedia(t *testing.T) {
	ctrl := gomock.NewController(t)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	sessionMedia := []*models.Media{
		mediaFixture("m1", "k1", 1000, 1, 2),
		mediaFixture("m2", "k2", 2000, 3, 4),
	}
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k1", mlPresignTTL).Return("https://signed/k1", nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k2", mlPresignTTL).Return("https://signed/k2", nil)

	msg, count, err := BuildMLTaskMessageForPinUpload(context.Background(),
		MLFlowPinUploadCreation, "trip-1", "sess-1", "", "", sessionMedia, nil, urls)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Equal(t, MLFlowPinUploadCreation, msg.Flow)
	require.Equal(t, "sess-1", msg.SessionID)
	require.Equal(t, "", msg.TargetPinID)
	require.Len(t, msg.Pins, 1)
	require.True(t, msg.Pins[0].IsNew)
	for _, m := range msg.Pins[0].Media {
		require.True(t, m.IsNew)
	}
}

func TestBuildMLTaskMessageForPinUpload_Addition_BothSets(t *testing.T) {
	ctrl := gomock.NewController(t)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	sessionMedia := []*models.Media{mediaFixture("new-1", "kn1", 100, 1, 2)}
	pinMedia := []*models.Media{
		mediaFixture("old-1", "ko1", 50, 5, 6),
		mediaFixture("old-2", "ko2", 60, 7, 8),
	}
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "kn1", mlPresignTTL).Return("https://signed/new", nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "ko1", mlPresignTTL).Return("https://signed/old1", nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "ko2", mlPresignTTL).Return("https://signed/old2", nil)

	msg, count, err := BuildMLTaskMessageForPinUpload(context.Background(),
		MLFlowPinUploadAddition, "trip-1", "sess-1", "pin-target", "", sessionMedia, pinMedia, urls)
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Equal(t, "pin-target", msg.TargetPinID)
	require.False(t, msg.Pins[0].IsNew)

	byID := make(map[string]repositories.MLMediaPayload, len(msg.Pins[0].Media))
	for _, m := range msg.Pins[0].Media {
		byID[m.MediaID] = m
	}
	require.True(t, byID["new-1"].IsNew)
	require.False(t, byID["old-1"].IsNew)
	require.False(t, byID["old-2"].IsNew)
}

func TestBuildMLTaskMessageForPinUpload_PresignError_Propagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k1", mlPresignTTL).Return("", errors.New("boom"))

	_, _, err := BuildMLTaskMessageForPinUpload(context.Background(),
		MLFlowPinUploadCreation, "trip-1", "sess-1", "", "",
		[]*models.Media{mediaFixture("m1", "k1", 0, 0, 0)}, nil, urls)
	require.Error(t, err)
}
