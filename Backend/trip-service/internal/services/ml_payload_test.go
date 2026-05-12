package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
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

func TestBuildTripMLPayload_Creation_AllPinsNew(t *testing.T) {
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

	pinsJSON, expiresAt, count, err := BuildTripMLPayload(context.Background(),
		"trip-1", mlFlowCreation, nil, nil, pinRepo, mediaRepo, urls)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Greater(t, expiresAt, time.Now().Unix())

	var pins []MLPinPayload
	require.NoError(t, json.Unmarshal([]byte(pinsJSON), &pins))
	require.Len(t, pins, 2)
	require.True(t, pins[0].IsNew)
	require.True(t, pins[1].IsNew)
	require.Equal(t, "https://signed/k1", pins[0].Media[0].GetURL)
	require.Equal(t, int64(1000), *pins[0].Media[0].CapturedAtUnix)
}

func TestBuildTripMLPayload_AddMedia_ExistingPinFilteredToPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	newPin := &models.Pin{ID: "pin-new", TripID: "trip-1"}
	existingPin := &models.Pin{ID: "pin-existing", TripID: "trip-1"}
	pinRepo.EXPECT().ListByTripID("trip-1").Return([]*models.Pin{newPin, existingPin}, nil)

	// New pin: всё медиа уезжает в payload.
	mediaRepo.EXPECT().ListByPinID("pin-new").Return([]*models.Media{
		mediaFixture("m-new-1", "k-new-1", 1000, 1, 2),
	}, nil)
	// Existing pin: в payload только новые медиа (m-added-1), исходное m-orig-1 фильтруется.
	mediaRepo.EXPECT().ListByPinID("pin-existing").Return([]*models.Media{
		mediaFixture("m-orig-1", "k-orig-1", 500, 5, 6),
		mediaFixture("m-added-1", "k-added-1", 2000, 7, 8),
	}, nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k-new-1", mlPresignTTL).Return("https://signed/new", nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k-added-1", mlPresignTTL).Return("https://signed/added", nil)

	pinsJSON, _, count, err := BuildTripMLPayload(context.Background(),
		"trip-1", mlFlowAddMedia,
		[]string{"pin-new"},
		[]string{"m-added-1"},
		pinRepo, mediaRepo, urls)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	var pins []MLPinPayload
	require.NoError(t, json.Unmarshal([]byte(pinsJSON), &pins))
	require.Len(t, pins, 2)

	byID := map[string]MLPinPayload{pins[0].PinID: pins[0], pins[1].PinID: pins[1]}
	require.True(t, byID["pin-new"].IsNew)
	require.False(t, byID["pin-existing"].IsNew)
	require.Len(t, byID["pin-existing"].Media, 1)
	require.Equal(t, "m-added-1", byID["pin-existing"].Media[0].MediaID)
}

func TestBuildTripMLPayload_AddMedia_PinWithoutPendingExcluded(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	existingPin := &models.Pin{ID: "pin-untouched", TripID: "trip-1"}
	pinRepo.EXPECT().ListByTripID("trip-1").Return([]*models.Pin{existingPin}, nil)
	mediaRepo.EXPECT().ListByPinID("pin-untouched").Return([]*models.Media{
		mediaFixture("m1", "k1", 0, 0, 0),
	}, nil)
	// urls не вызывается — медиа отфильтровано.

	pinsJSON, _, count, err := BuildTripMLPayload(context.Background(),
		"trip-1", mlFlowAddMedia, nil, nil, pinRepo, mediaRepo, urls)
	require.NoError(t, err)
	require.Equal(t, 0, count)
	require.Equal(t, "[]", pinsJSON)
}

func TestBuildTripMLPayload_NoCoordsNoCaptured_OmitFields(t *testing.T) {
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

	pinsJSON, _, _, err := BuildTripMLPayload(context.Background(),
		"trip-1", mlFlowCreation, nil, nil, pinRepo, mediaRepo, urls)
	require.NoError(t, err)
	require.NotContains(t, pinsJSON, `"latitude"`)
	require.NotContains(t, pinsJSON, `"captured_at_unix"`)
}

func TestBuildTripMLPayload_UnknownFlow_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	_, _, _, err := BuildTripMLPayload(context.Background(),
		"trip-1", "bogus", nil, nil, pinRepo, mediaRepo, urls)
	require.Error(t, err)
}

func TestBuildTripMLPayload_PresignError_Propagated(t *testing.T) {
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

	_, _, _, err := BuildTripMLPayload(context.Background(),
		"trip-1", mlFlowCreation, nil, nil, pinRepo, mediaRepo, urls)
	require.Error(t, err)
}

func TestBuildPinUploadMLPayload_Creation_OnlyNewMedia(t *testing.T) {
	ctrl := gomock.NewController(t)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	sessionMedia := []*models.Media{
		mediaFixture("m1", "k1", 1000, 1, 2),
		mediaFixture("m2", "k2", 2000, 3, 4),
	}
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k1", mlPresignTTL).Return("https://signed/k1", nil)
	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k2", mlPresignTTL).Return("https://signed/k2", nil)

	newJSON, existingJSON, _, count, err := BuildPinUploadMLPayload(context.Background(),
		MLFlowPinUploadCreate, sessionMedia, nil, urls)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Equal(t, "", existingJSON)

	var items []MLMediaPayload
	require.NoError(t, json.Unmarshal([]byte(newJSON), &items))
	require.Len(t, items, 2)
	require.Equal(t, "https://signed/k1", items[0].GetURL)
}

func TestBuildPinUploadMLPayload_Addition_BothSets(t *testing.T) {
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

	newJSON, existingJSON, _, count, err := BuildPinUploadMLPayload(context.Background(),
		MLFlowPinUploadAddTo, sessionMedia, pinMedia, urls)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	var newItems, oldItems []MLMediaPayload
	require.NoError(t, json.Unmarshal([]byte(newJSON), &newItems))
	require.NoError(t, json.Unmarshal([]byte(existingJSON), &oldItems))
	require.Len(t, newItems, 1)
	require.Len(t, oldItems, 2)
}

func TestBuildPinUploadMLPayload_PresignError_Propagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	urls.EXPECT().ReadURLWithTTL(gomock.Any(), "k1", mlPresignTTL).Return("", errors.New("boom"))

	_, _, _, _, err := BuildPinUploadMLPayload(context.Background(),
		MLFlowPinUploadCreate, []*models.Media{mediaFixture("m1", "k1", 0, 0, 0)}, nil, urls)
	require.Error(t, err)
}
