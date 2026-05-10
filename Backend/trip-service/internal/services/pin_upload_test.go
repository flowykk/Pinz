package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/repositories"
	pb "pinz/backend/trip-service/pkg/proto"
)

// helper: TripService с pinUploadSessionRepo на нужном месте.
func newSvcWithUploadRepo(
	tripRepo repositories.TripRepositoryInterface,
	participantRepo repositories.TripParticipantRepositoryInterface,
	mediaRepo repositories.MediaRepositoryInterface,
	pinRepo repositories.PinRepositoryInterface,
	tagRepo repositories.TagRepositoryInterface,
	favouriteRepo repositories.FavouriteRepositoryInterface,
	pinHiddenRepo repositories.PinHiddenRepositoryInterface,
	eventRepo repositories.TripEventPublisher,
	pinUploadRepo repositories.PinUploadSessionRepositoryInterface,
) *TripService {
	return NewTripService(
		tripRepo, participantRepo, nil, nil, eventRepo,
		mediaRepo, nil, pinRepo, tagRepo, nil, favouriteRepo,
		nil, nil, nil, nil, nil, nil,
		pinHiddenRepo, pinUploadRepo, nil,
	)
}

// =============================================================================
// PinUploadStart
// =============================================================================

func TestPinUploadStart_Creation_Success(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	mediaRepo.EXPECT().CountByTripID(tripID).Return(0, 0, nil)
	uploadRepo.EXPECT().Create(gomock.Any(), tripID, (*string)(nil), userID).Return("sess-1", nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, nil, nil, nil, nil, nil, uploadRepo)
	resp, err := svc.PinUploadStart(ctxWithUser(userID), &pb.PinUploadStartRequest{
		TripId:        tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.NoError(t, err)
	require.Equal(t, "sess-1", resp.GetSessionId())
}

func TestPinUploadStart_Addition_Success(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil)
	mediaRepo.EXPECT().CountByTripID(tripID).Return(0, 0, nil)
	pinTarget := pinID
	uploadRepo.EXPECT().Create(gomock.Any(), tripID, &pinTarget, userID).Return("sess-2", nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, pinRepo, nil, nil, nil, nil, uploadRepo)
	resp, err := svc.PinUploadStart(ctxWithUser(userID), &pb.PinUploadStartRequest{
		TripId:        tripID,
		TargetPinId:   &pinTarget,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.NoError(t, err)
	require.Equal(t, "sess-2", resp.GetSessionId())
}

func TestPinUploadStart_NotParticipant_PermissionDenied(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(false, nil)

	svc := newSvcWithUploadRepo(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.PinUploadStart(ctxWithUser(userID), &pb.PinUploadStartRequest{
		TripId:        tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestPinUploadStart_TripNotReady_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusAddMediaUploading}, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.PinUploadStart(ctxWithUser(userID), &pb.PinUploadStartRequest{
		TripId:        tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestPinUploadStart_DuplicateCreation_Conflict(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	mediaRepo.EXPECT().CountByTripID(tripID).Return(0, 0, nil)
	uploadRepo.EXPECT().Create(gomock.Any(), tripID, (*string)(nil), userID).Return("", repositories.ErrPinUploadSessionActive)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, nil, nil, nil, nil, nil, uploadRepo)
	_, err := svc.PinUploadStart(ctxWithUser(userID), &pb.PinUploadStartRequest{
		TripId:        tripID,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/jpeg"}},
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// =============================================================================
// ProcessPinUpload
// =============================================================================

func TestProcessPinUpload_TransitionsToProcessing(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusUploading,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	uploadRepo.EXPECT().SetProcessingStatus(gomock.Any(), sessionID,
		models.PinUploadProcessingStatusUploading,
		models.PinUploadProcessingStatusProcessing).Return(nil)
	eventRepo.EXPECT().AddPinUploadTask(gomock.Any(), tripID, sessionID, (*string)(nil), userID).Return(nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, nil, nil, nil, nil, nil, eventRepo, uploadRepo)
	resp, err := svc.ProcessPinUpload(ctxWithUser(userID), &pb.ProcessPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, models.PinUploadProcessingStatusProcessing, resp.GetProcessingStatus())
}

func TestProcessPinUpload_RejectsRepeatOnNonUploadingState(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusProcessing,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, uploadRepo)
	_, err := svc.ProcessPinUpload(ctxWithUser(userID), &pb.ProcessPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestProcessPinUpload_PublishFailure_LeavesProcessingState(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusUploading,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	uploadRepo.EXPECT().SetProcessingStatus(gomock.Any(), sessionID,
		models.PinUploadProcessingStatusUploading,
		models.PinUploadProcessingStatusProcessing).Return(nil)
	eventRepo.EXPECT().AddPinUploadTask(gomock.Any(), tripID, sessionID, (*string)(nil), userID).
		Return(errors.New("redis down"))

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, nil, nil, nil, nil, nil, eventRepo, uploadRepo)
	_, err := svc.ProcessPinUpload(ctxWithUser(userID), &pb.ProcessPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.Equal(t, codes.Internal, status.Code(err))
}

// =============================================================================
// FinalizePinUpload
// =============================================================================

func TestFinalizePinUpload_BeforeReadyForReview_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusUploading,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, uploadRepo)
	_, err := svc.FinalizePinUpload(ctxWithUser(userID), &pb.FinalizePinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// =============================================================================
// CancelPinUpload
// =============================================================================

func TestCancelPinUpload_RemovesOrphans(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusUploading,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	mediaRepo.EXPECT().DeleteOrphanByUploadSession(sessionID).Return([]string{"k1"}, nil)
	uploadRepo.EXPECT().Close(gomock.Any(), sessionID, models.PinUploadSessionCloseReasonCancelled).Return(nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, nil, nil, nil, nil, nil, uploadRepo)
	resp, err := svc.CancelPinUpload(ctxWithUser(userID), &pb.CancelPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, "cancelled", resp.GetStatus())
}

func TestCancelPinUpload_OnClosedSession_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)
	closedAt := time.Now().Add(-time.Hour)
	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusReadyForReview,
		ClosedAt:         &closedAt,
	}, nil)

	svc := newSvcWithUploadRepo(nil, nil, nil, nil, nil, nil, nil, nil, uploadRepo)
	_, err := svc.CancelPinUpload(ctxWithUser(userID), &pb.CancelPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// =============================================================================
// PinUploadStart — UNIQUE corner cases
// =============================================================================

func TestPinUploadStart_DuplicateAdditionSamePin_Conflict(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil)
	mediaRepo.EXPECT().CountByTripID(tripID).Return(0, 0, nil)
	target := pinID
	uploadRepo.EXPECT().Create(gomock.Any(), tripID, &target, userID).Return("", repositories.ErrPinUploadSessionActive)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, pinRepo, nil, nil, nil, nil, uploadRepo)
	_, err := svc.PinUploadStart(ctxWithUser(userID), &pb.PinUploadStartRequest{
		TripId: tripID, TargetPinId: &target,
		FilesToUpload: []*pb.FileToUpload{{ClientId: "x", ContentType: "image/jpeg"}},
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// =============================================================================
// CommitPinUpload
// =============================================================================

func TestCommitPinUpload_Success_LimitsRespected(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusUploading,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady, PrivacyLevel: "Private"}, nil).Times(2)
	mediaRepo.EXPECT().CommitInUploadSession(gomock.Any(), gomock.Any(), sessionID, MaxMediaPerTrip, MaxVideosPerTrip).
		DoAndReturn(func(_ context.Context, m *models.Media, _ string, _, _ int) (int, int, error) {
			m.ID = "media-new"
			require.NotNil(t, m.UploadSessionID)
			require.Equal(t, sessionID, *m.UploadSessionID)
			require.Nil(t, m.PinID)
			return 6, 1, nil
		})
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{{ID: "media-new"}}, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, nil, nil, nil, nil, nil, uploadRepo)
	resp, err := svc.CommitPinUpload(ctxWithUser(userID), &pb.CommitPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
		S3Key: "trips/t1/pins/x.jpg", MediaType: "image",
	})
	require.NoError(t, err)
	require.Equal(t, "media-new", resp.GetMediaId())
	require.Equal(t, int32(1), resp.GetMediaCountInSession())
}

func TestCommitPinUpload_OverMediaLimit_ResourceExhausted(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusUploading,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil).Times(2)
	mediaRepo.EXPECT().CommitInUploadSession(gomock.Any(), gomock.Any(), sessionID, MaxMediaPerTrip, MaxVideosPerTrip).
		Return(MaxMediaPerTrip, 0, repositories.ErrMediaLimitExceeded)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, nil, nil, nil, nil, nil, uploadRepo)
	_, err := svc.CommitPinUpload(ctxWithUser(userID), &pb.CommitPinUploadRequest{
		TripId: tripID, SessionId: sessionID,
		S3Key: "trips/t1/pins/x.jpg", MediaType: "image",
	})
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
}

// =============================================================================
// GetPinUploadReview
// =============================================================================

func TestGetPinUploadReview_BeforeProcessing_EmptyDraft(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusUploading,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, uploadRepo)
	resp, err := svc.GetPinUploadReview(ctxWithUser(userID), &pb.GetPinUploadReviewRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, models.PinUploadProcessingStatusUploading, resp.GetProcessingStatus())
	require.Nil(t, resp.GetDraft())
}

func TestGetPinUploadReview_ReadyForReview_FullDraftCreation(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	lat := 59.9
	lon := 30.3
	start := int64(1000)
	end := int64(2000)
	snap := pinUploadDraftSnapshot{
		Suggested: &pinSuggestedFields{
			Name: "Другое", Category: "Другое",
			Latitude: &lat, Longitude: &lon,
			StartTimeUnix: &start, EndTimeUnix: &end,
		},
		NewMediaIDs: []string{"m1"},
	}
	snapBytes, _ := json.Marshal(snap)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusReadyForReview,
		DraftSnapshot:    snapBytes,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{
		{ID: "m1", S3Key: "k1", PrivacyLevel: "Private"},
	}, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, nil, nil, nil, nil, nil, uploadRepo)
	resp, err := svc.GetPinUploadReview(ctxWithUser(userID), &pb.GetPinUploadReviewRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, models.PinUploadProcessingStatusReadyForReview, resp.GetProcessingStatus())
	require.NotNil(t, resp.GetDraft())
	require.NotNil(t, resp.GetDraft().GetSuggested())
	require.Equal(t, "Другое", resp.GetDraft().GetSuggested().GetCategory())
	require.Len(t, resp.GetDraft().GetMedia(), 1)
}

func TestGetPinUploadReview_ReadyForReview_AdditionHasNoSuggested(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	snap := pinUploadDraftSnapshot{NewMediaIDs: []string{"m1"}}
	snapBytes, _ := json.Marshal(snap)

	target := pinID
	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, TargetPinID: &target, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusReadyForReview,
		DraftSnapshot:    snapBytes,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID}, nil)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{{ID: "m1", S3Key: "k1"}}, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, pinRepo, nil, nil, nil, nil, uploadRepo)
	resp, err := svc.GetPinUploadReview(ctxWithUser(userID), &pb.GetPinUploadReviewRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, models.PinUploadProcessingStatusReadyForReview, resp.GetProcessingStatus())
	require.NotNil(t, resp.GetDraft())
	require.Nil(t, resp.GetDraft().GetSuggested(), "addition: suggested обязан быть nil")
	require.Len(t, resp.GetDraft().GetMedia(), 1)
}

// =============================================================================
// FinalizePinUpload — happy-path
// =============================================================================

func TestFinalizePinUpload_Creation_HappyPath_CreatesPinAndPublishesEvent(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	snap := pinUploadDraftSnapshot{
		Suggested: &pinSuggestedFields{
			Name: "Другое", Category: "Другое", Tags: []string{},
		},
		NewMediaIDs: []string{"m1", "m2"},
	}
	snapBytes, _ := json.Marshal(snap)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusReadyForReview,
		DraftSnapshot:    snapBytes,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady, PrivacyLevel: "Private"}, nil).Times(2)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{
		{ID: "m1", S3Key: "k1"},
		{ID: "m2", S3Key: "k2"},
	}, nil)

	pinRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(p *models.Pin) error {
		p.ID = "pin-new"
		require.Equal(t, "Кафе", p.Name)
		require.Equal(t, int32(2), p.MediaCount)
		return nil
	})
	mediaRepo.EXPECT().UpdatePinIDByIDs([]string{"m1", "m2"}, "pin-new").Return(nil)
	tagRepo.EXPECT().SetForPin(tripID, "pin-new", []string{"food"}).Return(nil)
	mediaRepo.EXPECT().ListByPinID("pin-new").Return([]*models.Media{
		{ID: "m1"}, {ID: "m2"},
	}, nil).Times(2)
	pinRepo.EXPECT().GetByID("pin-new").Return(&models.Pin{ID: "pin-new", TripID: tripID, Name: "Кафе"}, nil).Times(2)
	pinRepo.EXPECT().Update(gomock.Any()).Return(nil)

	eventRepo.EXPECT().PublishTripEvent(gomock.Any(), "PIN_ADDED", tripID, userID).Return(nil)
	uploadRepo.EXPECT().Close(gomock.Any(), sessionID, models.PinUploadSessionCloseReasonConfirmed).Return(nil)
	tagRepo.EXPECT().GetByPinID("pin-new").Return([]string{"food"}, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, pinRepo, tagRepo, nil, nil, eventRepo, uploadRepo)
	name := "Кафе"
	resp, err := svc.FinalizePinUpload(ctxWithUser(userID), &pb.FinalizePinUploadRequest{
		TripId: tripID, SessionId: sessionID,
		Name: &name, Tags: []string{"food"}, TagsSet: true,
	})
	require.NoError(t, err)
	require.Equal(t, "pin-new", resp.GetPin().GetId())
	require.Equal(t, "Кафе", resp.GetPin().GetName())
}

func TestFinalizePinUpload_Addition_HappyPath_AttachesMediaAndIncrements(t *testing.T) {
	const tripID = "trip-1"
	const pinID = "pin-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	target := pinID
	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, TargetPinID: &target, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusReadyForReview,
		DraftSnapshot:    []byte(`{"new_media_ids":["m1"]}`),
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil).Times(2)
	pinRepo.EXPECT().GetByID(pinID).Return(&models.Pin{ID: pinID, TripID: tripID, Name: "Old", MediaCount: 1}, nil).Times(3)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return([]*models.Media{{ID: "m1", S3Key: "k1"}}, nil)
	mediaRepo.EXPECT().UpdatePinIDByIDs([]string{"m1"}, pinID).Return(nil)
	pinRepo.EXPECT().IncMediaCount(pinID, 1).Return(nil)
	mediaRepo.EXPECT().ListByPinID(pinID).Return([]*models.Media{{ID: "m1"}}, nil).Times(2)
	pinRepo.EXPECT().Update(gomock.Any()).Return(nil)
	uploadRepo.EXPECT().Close(gomock.Any(), sessionID, models.PinUploadSessionCloseReasonConfirmed).Return(nil)
	tagRepo.EXPECT().GetByPinID(pinID).Return(nil, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, pinRepo, tagRepo, nil, nil, nil, uploadRepo)
	resp, err := svc.FinalizePinUpload(ctxWithUser(userID), &pb.FinalizePinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Equal(t, pinID, resp.GetPin().GetId())
}

func TestFinalizePinUpload_Creation_EmptyRemainingMedia_FailedPrecondition(t *testing.T) {
	const tripID = "trip-1"
	const sessionID = "sess-1"
	const userID = "user-1"
	ctrl := gomock.NewController(t)

	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	uploadRepo := mocks.NewMockPinUploadSessionRepositoryInterface(ctrl)

	uploadRepo.EXPECT().GetByID(gomock.Any(), sessionID).Return(&models.PinUploadSession{
		SessionID: sessionID, TripID: tripID, InitiatorUserID: userID,
		ProcessingStatus: models.PinUploadProcessingStatusReadyForReview,
	}, nil)
	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(&models.Trip{ID: tripID, Status: models.TripStatusReady}, nil).Times(2)
	mediaRepo.EXPECT().ListByUploadSession(sessionID).Return(nil, nil)

	svc := newSvcWithUploadRepo(tripRepo, participantRepo, mediaRepo, nil, nil, nil, nil, nil, uploadRepo)
	_, err := svc.FinalizePinUpload(ctxWithUser(userID), &pb.FinalizePinUploadRequest{
		TripId: tripID, SessionId: sessionID,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}
