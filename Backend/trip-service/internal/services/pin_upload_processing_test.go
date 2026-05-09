package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
)

// mediaRepoForProcessingShim адаптирует MockMediaRepositoryInterface к узкому
// MediaRepositoryForUploadProcessing (нужно из-за отдельного интерфейса в pin_upload_processing.go).
type mediaRepoForProcessingShim struct {
	*mocks.MockMediaRepositoryInterface
}

func TestRunPinUploadProcessing_Creation_HashDedupAndSuggested(t *testing.T) {
	const sessionID = "sess-1"
	ctrl := gomock.NewController(t)

	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TargetPinID: nil,
		ProcessingStatus: models.PinUploadProcessingStatusProcessing,
	}, nil)

	hashA := "hash-A"
	hashB := "hash-B"
	lat := 59.9
	lon := 30.3
	captured1 := time.Unix(1000, 0)
	captured2 := time.Unix(2000, 0)
	sessionMedia := []*models.Media{
		{ID: "m1", S3Key: "k1", ContentHash: &hashA, Latitude: &lat, Longitude: &lon, CapturedAt: &captured1},
		{ID: "m2", S3Key: "k2", ContentHash: &hashA, CapturedAt: &captured2}, // дубликат m1
		{ID: "m3", S3Key: "k3", ContentHash: &hashB},
	}
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return(sessionMedia, nil)
	mediaRepo.EXPECT().DeleteByIDs([]string{"m2"}).Return(nil)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{
		{ID: "m1", S3Key: "k1", ContentHash: &hashA, Latitude: &lat, Longitude: &lon, CapturedAt: &captured1},
		{ID: "m3", S3Key: "k3", ContentHash: &hashB},
	}, nil)

	var capturedSnap []byte
	uploadRepo.EXPECT().SetDraftSnapshot(gomock.Any(), sessionID, gomock.Any()).
		DoAndReturn(func(_ any, _ string, snap []byte) error {
			capturedSnap = snap
			return nil
		})
	uploadRepo.EXPECT().SetProcessingStatus(gomock.Any(), sessionID,
		models.PinUploadProcessingStatusProcessing,
		models.PinUploadProcessingStatusReadyForReview).Return(nil)

	transitioned, err := RunPinUploadProcessing(context.Background(), sessionID, PinUploadProcessorDeps{
		SessionRepo: uploadRepo,
		MediaRepo:   &mediaRepoForProcessingShim{mediaRepo},
	})
	require.NoError(t, err)
	require.True(t, transitioned)

	var snap pinUploadDraftSnapshot
	require.NoError(t, json.Unmarshal(capturedSnap, &snap))
	require.NotNil(t, snap.Suggested, "creation: suggested обязан быть")
	require.Equal(t, PinCategoryDefault, snap.Suggested.Category)
	require.NotNil(t, snap.Suggested.Latitude)
	require.Equal(t, lat, *snap.Suggested.Latitude)
	require.Equal(t, []string{"m2"}, snap.DedupedMediaIDs)
	require.NotContains(t, snap.PinIssues, pinIssueMissingCoordinates)
	require.NotContains(t, snap.PinIssues, pinIssueMissingDates)
}

func TestRunPinUploadProcessing_Creation_PinIssues_NoCoordsNoDates(t *testing.T) {
	const sessionID = "sess-1"
	ctrl := gomock.NewController(t)

	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TargetPinID: nil,
		ProcessingStatus: models.PinUploadProcessingStatusProcessing,
	}, nil)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{{ID: "m1", S3Key: "k1"}}, nil).Times(2)

	var capturedSnap []byte
	uploadRepo.EXPECT().SetDraftSnapshot(gomock.Any(), sessionID, gomock.Any()).
		DoAndReturn(func(_ any, _ string, snap []byte) error {
			capturedSnap = snap
			return nil
		})
	uploadRepo.EXPECT().SetProcessingStatus(gomock.Any(), sessionID,
		models.PinUploadProcessingStatusProcessing,
		models.PinUploadProcessingStatusReadyForReview).Return(nil)

	transitioned, err := RunPinUploadProcessing(context.Background(), sessionID, PinUploadProcessorDeps{
		SessionRepo: uploadRepo,
		MediaRepo:   &mediaRepoForProcessingShim{mediaRepo},
	})
	require.NoError(t, err)
	require.True(t, transitioned)

	var snap pinUploadDraftSnapshot
	require.NoError(t, json.Unmarshal(capturedSnap, &snap))
	require.Contains(t, snap.PinIssues, pinIssueMissingCoordinates)
	require.Contains(t, snap.PinIssues, pinIssueMissingDates)
}

func TestRunPinUploadProcessing_Addition_DedupAgainstExistingPinMedia(t *testing.T) {
	const sessionID = "sess-1"
	const pinID = "pin-1"
	ctrl := gomock.NewController(t)

	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	target := pinID
	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TargetPinID: &target,
		ProcessingStatus: models.PinUploadProcessingStatusProcessing,
	}, nil)

	hashShared := "hash-shared"
	hashUnique := "hash-unique"
	captured := time.Unix(1000, 0)
	lat := 59.9
	lon := 30.3
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{
		{ID: "new1", S3Key: "k1", ContentHash: &hashShared}, // совпадает с существующим pin.media
		{ID: "new2", S3Key: "k2", ContentHash: &hashUnique, CapturedAt: &captured, Latitude: &lat, Longitude: &lon},
	}, nil)
	mediaRepo.EXPECT().ListByPinID(pinID).Return([]*models.Media{
		{ID: "old1", ContentHash: &hashShared, CapturedAt: &captured, Latitude: &lat, Longitude: &lon},
	}, nil)
	mediaRepo.EXPECT().DeleteByIDs([]string{"new1"}).Return(nil)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{
		{ID: "new2", S3Key: "k2", ContentHash: &hashUnique, CapturedAt: &captured, Latitude: &lat, Longitude: &lon},
	}, nil)

	var capturedSnap []byte
	uploadRepo.EXPECT().SetDraftSnapshot(gomock.Any(), sessionID, gomock.Any()).
		DoAndReturn(func(_ any, _ string, snap []byte) error {
			capturedSnap = snap
			return nil
		})
	uploadRepo.EXPECT().SetProcessingStatus(gomock.Any(), sessionID,
		models.PinUploadProcessingStatusProcessing,
		models.PinUploadProcessingStatusReadyForReview).Return(nil)

	transitioned, err := RunPinUploadProcessing(context.Background(), sessionID, PinUploadProcessorDeps{
		SessionRepo: uploadRepo,
		MediaRepo:   &mediaRepoForProcessingShim{mediaRepo},
	})
	require.NoError(t, err)
	require.True(t, transitioned)

	var snap pinUploadDraftSnapshot
	require.NoError(t, json.Unmarshal(capturedSnap, &snap))
	require.Nil(t, snap.Suggested, "addition: suggested обязан быть nil")
	require.Equal(t, []string{"new1"}, snap.DedupedMediaIDs)
	require.Equal(t, []string{"new2"}, snap.NewMediaIDs)
	// Координаты и даты есть у remaining + pin.media → issues пустые.
	require.NotContains(t, snap.PinIssues, pinIssueMissingCoordinates)
	require.NotContains(t, snap.PinIssues, pinIssueMissingDates)
}

func TestRunPinUploadProcessing_Addition_PinIssuesByCombinedSet(t *testing.T) {
	const sessionID = "sess-1"
	const pinID = "pin-1"
	ctrl := gomock.NewController(t)

	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	target := pinID
	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TargetPinID: &target,
		ProcessingStatus: models.PinUploadProcessingStatusProcessing,
	}, nil)
	// session media без координат и без captured_at, pin media — то же.
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{{ID: "new1", S3Key: "k1"}}, nil).Times(2)
	mediaRepo.EXPECT().ListByPinID(pinID).Return([]*models.Media{{ID: "old1"}}, nil)

	var capturedSnap []byte
	uploadRepo.EXPECT().SetDraftSnapshot(gomock.Any(), sessionID, gomock.Any()).
		DoAndReturn(func(_ any, _ string, snap []byte) error {
			capturedSnap = snap
			return nil
		})
	uploadRepo.EXPECT().SetProcessingStatus(gomock.Any(), sessionID,
		models.PinUploadProcessingStatusProcessing,
		models.PinUploadProcessingStatusReadyForReview).Return(nil)

	transitioned, err := RunPinUploadProcessing(context.Background(), sessionID, PinUploadProcessorDeps{
		SessionRepo: uploadRepo,
		MediaRepo:   &mediaRepoForProcessingShim{mediaRepo},
	})
	require.NoError(t, err)
	require.True(t, transitioned)

	var snap pinUploadDraftSnapshot
	require.NoError(t, json.Unmarshal(capturedSnap, &snap))
	require.Contains(t, snap.PinIssues, pinIssueMissingCoordinates)
	require.Contains(t, snap.PinIssues, pinIssueMissingDates)
}

func TestRunPinUploadProcessing_SessionAlreadyClosed_NoOp(t *testing.T) {
	const sessionID = "sess-1"
	ctrl := gomock.NewController(t)

	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID:        sessionID,
		ProcessingStatus: models.PinUploadProcessingStatusProcessing,
	}, nil)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return(nil, nil).Times(2)
	uploadRepo.EXPECT().SetDraftSnapshot(gomock.Any(), sessionID, gomock.Any()).Return(nil)
	// CAS не сработал — сессия уже закрыта.
	uploadRepo.EXPECT().SetProcessingStatus(gomock.Any(), sessionID,
		models.PinUploadProcessingStatusProcessing,
		models.PinUploadProcessingStatusReadyForReview).Return(repositories.ErrPinUploadSessionWrongState)

	transitioned, err := RunPinUploadProcessing(context.Background(), sessionID, PinUploadProcessorDeps{
		SessionRepo: uploadRepo,
		MediaRepo:   &mediaRepoForProcessingShim{mediaRepo},
	})
	require.NoError(t, err, "wrong-state на CAS — не ошибка, а корректный noop")
	require.False(t, transitioned, "wrong-state на CAS → transitioned=false")
}
