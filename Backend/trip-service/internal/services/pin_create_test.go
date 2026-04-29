package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	pb "pinz/backend/trip-service/pkg/proto"
)

// =============================================================================
// CreatePinStart
// =============================================================================

func TestCreatePinStart_NotParticipant_PermissionDenied(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(false, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.CreatePinStart(ctxWithUser(userID), &pb.CreatePinStartRequest{
		TripId: tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCreatePinStart_TripNotReady_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusAddMediaUploading}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.CreatePinStart(ctxWithUser(userID), &pb.CreatePinStartRequest{
		TripId: tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestCreatePinStart_LimitExceeded_ResourceExhausted(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	mediaRepo.EXPECT().CountByTripID(tripID).Return(MaxMediaPerTrip, 0, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.CreatePinStart(ctxWithUser(userID), &pb.CreatePinStartRequest{
		TripId: tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestCreatePinStart_ConflictExistingSession_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockPinCreationSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	mediaRepo.EXPECT().CountByTripID(tripID).Return(0, 0, nil)
	sessionRepo.EXPECT().Create(gomock.Any(), tripID, userID).Return("", repositories.ErrPinCreationSessionActive)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sessionRepo)
	_, err := svc.CreatePinStart(ctxWithUser(userID), &pb.CreatePinStartRequest{
		TripId: tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// =============================================================================
// CommitPinCreationUpload
// =============================================================================

func TestCommitPinCreationUpload_Success(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockPinCreationSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady, PrivacyLevel: "Private"}, nil).Times(2)
	sessionRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinCreationSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
	}, nil)
	mediaRepo.EXPECT().CountByTripID(tripID).Return(5, 0, nil)
	mediaRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(m *models.Media) error {
		m.ID = "media-new"
		require.NotNil(t, m.PinCreationSessionID)
		require.Equal(t, sessionID, *m.PinCreationSessionID)
		require.Nil(t, m.PinID, "media must not be attached to any pin yet")
		return nil
	})
	sessionRepo.EXPECT().Touch(gomock.Any(), sessionID).Return(nil)
	mediaRepo.EXPECT().ListByPinCreationSession(sessionID).Return([]*models.Media{{ID: "media-new"}}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sessionRepo)
	resp, err := svc.CommitPinCreationUpload(ctxWithUser(userID), &pb.CommitPinCreationUploadRequest{
		TripId: tripID, SessionId: sessionID,
		S3Key: "trips/trip-1/pins/c1.jpg", MediaType: "image",
	})
	require.NoError(t, err)
	require.Equal(t, "media-new", resp.GetMediaId())
	require.Equal(t, int32(1), resp.GetMediaCountInSession())
}

// =============================================================================
// ProcessPinCreation
// =============================================================================

func TestProcessPinCreation_HashDedup_AndSuggestedFields(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockPinCreationSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady, Category: "Отпуск"}, nil).Times(2)
	sessionRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinCreationSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
	}, nil)

	expectedCategory := PinCategoryDefault
	hashA := "hash-A"
	hashB := "hash-B"
	lat1 := 55.0
	lon1 := 37.0
	mediaInitial := []*models.Media{
		{ID: "m1", S3Key: "k1", ContentHash: &hashA, Latitude: &lat1, Longitude: &lon1},
		{ID: "m2", S3Key: "k2", ContentHash: &hashA}, // дубликат m1
		{ID: "m3", S3Key: "k3", ContentHash: &hashB},
	}
	mediaRepo.EXPECT().ListByPinCreationSession(sessionID).Return(mediaInitial, nil)
	mediaRepo.EXPECT().DeleteByIDs([]string{"m2"}).Return(nil)
	mediaRepo.EXPECT().ListByPinCreationSession(sessionID).Return([]*models.Media{
		{ID: "m1", S3Key: "k1", ContentHash: &hashA, Latitude: &lat1, Longitude: &lon1},
		{ID: "m3", S3Key: "k3", ContentHash: &hashB},
	}, nil)
	var capturedSnapshot []byte
	sessionRepo.EXPECT().SetDraftSnapshot(gomock.Any(), sessionID, gomock.Any()).
		DoAndReturn(func(_ any, _ string, snap []byte) error {
			capturedSnapshot = snap
			return nil
		})

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sessionRepo)
	resp, err := svc.ProcessPinCreation(ctxWithUser(userID), &pb.ProcessPinCreationRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.GetDraft())
	require.Equal(t, expectedCategory, resp.GetDraft().GetSuggestedName(), "name = category по ТЗ 4.7.2.f")
	require.Equal(t, expectedCategory, resp.GetDraft().GetSuggestedCategory())
	require.NotNil(t, resp.GetDraft().SuggestedLatitude)
	require.Equal(t, lat1, *resp.GetDraft().SuggestedLatitude)
	require.Equal(t, []string{"m2"}, resp.GetDraft().GetDedupedMediaIds())
	// Snapshot validation.
	require.NotEmpty(t, capturedSnapshot)
	var snap pinCreationDraftSnapshot
	require.NoError(t, json.Unmarshal(capturedSnapshot, &snap))
	require.Equal(t, []string{"m2"}, snap.DedupedMediaIDs)
}

func TestProcessPinCreation_PinIssues_MissingCoordsAndDates(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockPinCreationSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady, Category: "Отпуск"}, nil).Times(2)
	sessionRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinCreationSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
	}, nil)
	// media без captured_at и координат → оба issues.
	media := []*models.Media{{ID: "m1", S3Key: "k1"}}
	mediaRepo.EXPECT().ListByPinCreationSession(sessionID).Return(media, nil).Times(2)
	sessionRepo.EXPECT().SetDraftSnapshot(gomock.Any(), sessionID, gomock.Any()).Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sessionRepo)
	resp, err := svc.ProcessPinCreation(ctxWithUser(userID), &pb.ProcessPinCreationRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	issues := resp.GetDraft().GetPinIssues()
	require.Contains(t, issues, pinIssueMissingCoordinates)
	require.Contains(t, issues, pinIssueMissingDates)
}

// =============================================================================
// FinalizePinCreation
// =============================================================================

func TestFinalizePinCreation_HappyPath_CreatesPinAndPublishesEvent(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockPinCreationSessionRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	// Snapshot: ML-stub возвращает PinCategoryDefault как suggested-категорию пина.
	snap := pinCreationDraftSnapshot{
		SuggestedName: PinCategoryDefault,
		SuggestedCategory: PinCategoryDefault,
		SuggestedTags: []string{},
		NewMediaIDs: []string{"m1", "m2"},
	}
	snapBytes, _ := json.Marshal(snap)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady, Category: "Отпуск", PrivacyLevel: "Private"}, nil).Times(2)
	sessionRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinCreationSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID, DraftSnapshot: snapBytes,
	}, nil)
	mediaRepo.EXPECT().ListByPinCreationSession(sessionID).Return([]*models.Media{
		{ID: "m1", S3Key: "k1"},
		{ID: "m2", S3Key: "k2"},
	}, nil)

	// Pin создан.
	pinRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(p *models.Pin) error {
		p.ID = "pin-new"
		require.Equal(t, "Кафе у моря", p.Name, "пользовательский name перекрывает suggested")
		require.Equal(t, PinCategoryDefault, p.Category)
		require.Equal(t, int32(2), p.MediaCount)
		return nil
	})
	mediaRepo.EXPECT().UpdatePinIDByIDs([]string{"m1", "m2"}, "pin-new").Return(nil)
	tagRepo.EXPECT().SetForPin(tripID, "pin-new", []string{"sea", "summer"}).Return(nil)
	// updatePinTimesAndLocation вызывается, т.к. lat/start/end в req не заданы.
	mediaRepo.EXPECT().ListByPinID("pin-new").Return([]*models.Media{
		{ID: "m1", S3Key: "k1"},
		{ID: "m2", S3Key: "k2"},
	}, nil).Times(2)
	pinRepo.EXPECT().GetByID("pin-new").Return(&models.Pin{ID: "pin-new", TripID: tripID, Name: "Кафе у моря"}, nil).Times(2)
	pinRepo.EXPECT().Update(gomock.Any()).Return(nil)
	// PIN_ADDED event.
	eventRepo.EXPECT().PublishTripEvent(gomock.Any(), "PIN_ADDED", tripID, userID).Return(nil)
	// Закрытие сессии.
	sessionRepo.EXPECT().Close(gomock.Any(), sessionID, models.PinCreationSessionCloseReasonConfirmed).Return(nil)
	// Финальный сбор.
	tagRepo.EXPECT().GetByPinID("pin-new").Return([]string{"sea", "summer"}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, mediaRepo, nil, pinRepo, tagRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sessionRepo)
	name := "Кафе у моря"
	resp, err := svc.FinalizePinCreation(ctxWithUser(userID), &pb.FinalizePinCreationRequest{
		TripId: tripID, SessionId: sessionID,
		Name: &name,
		Tags: []string{"sea", "summer"}, TagsSet: true,
	})
	require.NoError(t, err)
	require.Equal(t, "pin-new", resp.GetPin().GetId())
	require.Equal(t, "Кафе у моря", resp.GetPin().GetName())
}

func TestFinalizePinCreation_EmptyMedia_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockPinCreationSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil).Times(2)
	sessionRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinCreationSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
	}, nil)
	// Все media были удалены / нет.
	mediaRepo.EXPECT().ListByPinCreationSession(sessionID).Return(nil, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sessionRepo)
	_, err := svc.FinalizePinCreation(ctxWithUser(userID), &pb.FinalizePinCreationRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// =============================================================================
// CancelPinCreation
// =============================================================================

func TestCancelPinCreation_RemovesOrphans(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	sessionRepo := mocks.NewMockPinCreationSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	sessionRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinCreationSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
	}, nil)
	mediaRepo.EXPECT().DeleteOrphanByPinCreationSession(sessionID).Return([]string{"k1", "k2"}, nil)
	sessionRepo.EXPECT().Close(gomock.Any(), sessionID, models.PinCreationSessionCloseReasonCancelled).Return(nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, sessionRepo)
	resp, err := svc.CancelPinCreation(ctxWithUser(userID), &pb.CancelPinCreationRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, "cancelled", resp.GetStatus())
}
