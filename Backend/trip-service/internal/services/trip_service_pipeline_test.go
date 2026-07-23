package services

import (
	"context"
	"database/sql"
	"errors"
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

func TestProcessMediaGrouping_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ProcessMediaGrouping(context.Background(), &pb.ProcessMediaGroupingRequest{TripId: "t1"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestProcessMediaGrouping_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ProcessMediaGrouping(ctxWithUser("u1"), &pb.ProcessMediaGroupingRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestProcessMediaGrouping_TripNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(nil, sql.ErrNoRows)
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ProcessMediaGrouping(ctxWithUser("u1"), &pb.ProcessMediaGroupingRequest{TripId: "t1"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestProcessMediaGrouping_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: models.TripStatusDraft}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(false, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ProcessMediaGrouping(ctxWithUser("u1"), &pb.ProcessMediaGroupingRequest{TripId: "t1"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestProcessMediaGrouping_WrongStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: models.TripStatusReady}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ProcessMediaGrouping(ctxWithUser("u1"), &pb.ProcessMediaGroupingRequest{TripId: "t1"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestProcessMediaGrouping_MediaLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: models.TripStatusDraft}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	mediaRepo.EXPECT().CountByTripID("t1").Return(MaxMediaPerTrip, 0, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ProcessMediaGrouping(ctxWithUser("u1"), &pb.ProcessMediaGroupingRequest{
		TripId: "t1",
		Media:  []*pb.MediaMeta{{S3Key: "k", MediaType: "image"}},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestProcessMediaGrouping_VideoLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: models.TripStatusDraft}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	mediaRepo.EXPECT().CountByTripID("t1").Return(0, MaxVideosPerTrip, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ProcessMediaGrouping(ctxWithUser("u1"), &pb.ProcessMediaGroupingRequest{
		TripId: "t1",
		Media:  []*pb.MediaMeta{{S3Key: "k", MediaType: "video"}},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestProcessMediaGrouping_HappyPath_TransitionsStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: models.TripStatusDraft, PrivacyLevel: "private"}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	mediaRepo.EXPECT().CountByTripID("t1").Return(0, 0, nil)
	mediaRepo.EXPECT().CreateBatch(gomock.Any()).DoAndReturn(func(medias []*models.Media) error {
		for i := range medias {
			medias[i].ID = "m-new"
		}
		return nil
	})
	mediaRepo.EXPECT().ListByTripID("t1").Return([]*models.Media{
		{ID: "m-new", TripID: "t1", S3Key: "key1", MediaType: "image"},
	}, nil).Times(2)
	mediaRepo.EXPECT().ClusterIDsByLocation("t1", float64(ClusterRadiusMeters)).Return(map[string]int{}, nil)
	tripRepo.EXPECT().SetStatus("t1", "DRAFT_GROUPING_REVIEW").Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ProcessMediaGrouping(ctxWithUser("u1"), &pb.ProcessMediaGroupingRequest{
		TripId: "t1",
		Media:  []*pb.MediaMeta{{S3Key: "key1", MediaType: "image"}},
	})
	require.NoError(t, err)
	require.Equal(t, "DRAFT_GROUPING_REVIEW", resp.GetStatus())
}

func TestProcessMediaGrouping_MediaCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: models.TripStatusUploading}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	mediaRepo.EXPECT().CountByTripID("t1").Return(0, 0, nil)
	mediaRepo.EXPECT().CreateBatch(gomock.Any()).Return(errors.New("db down"))
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ProcessMediaGrouping(ctxWithUser("u1"), &pb.ProcessMediaGroupingRequest{
		TripId: "t1",
		Media:  []*pb.MediaMeta{{S3Key: "k", MediaType: "image"}},
	})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestApplyGroupsAndProcess_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ApplyGroupsAndProcess(context.Background(), &pb.ApplyGroupsAndProcessRequest{TripId: "t1"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestApplyGroupsAndProcess_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ApplyGroupsAndProcess(ctxWithUser("u1"), &pb.ApplyGroupsAndProcessRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestApplyGroupsAndProcess_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(false, nil)
	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ApplyGroupsAndProcess(ctxWithUser("u1"), &pb.ApplyGroupsAndProcessRequest{TripId: "t1"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestApplyGroupsAndProcess_TripNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(nil, sql.ErrNoRows)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ApplyGroupsAndProcess(ctxWithUser("u1"), &pb.ApplyGroupsAndProcessRequest{TripId: "t1"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestApplyGroupsAndProcess_WrongStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: models.TripStatusDraft}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ApplyGroupsAndProcess(ctxWithUser("u1"), &pb.ApplyGroupsAndProcessRequest{TripId: "t1"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestApplyGroupsAndProcess_HappyPath_NoDraftPins(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: "DRAFT_GROUPING_REVIEW", Category: "vacation", PrivacyLevel: "private"}, nil)
	tripRepo.EXPECT().SetStatus("t1", models.TripStatusProcessing).Return(nil)
	tripRepo.EXPECT().SetStatus("t1", models.TripStatusDraftFinalReview).Return(nil)
	eventRepo.EXPECT().DeleteTripEventStream(gomock.Any(), "t1").Return(nil)
	eventRepo.EXPECT().PublishTripEventWS(gomock.Any(), "t1", repositories.EventTripProcessingCompleted, gomock.Any()).Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ApplyGroupsAndProcess(ctxWithUser("u1"), &pb.ApplyGroupsAndProcessRequest{TripId: "t1"})
	require.NoError(t, err)
	require.Equal(t, models.TripStatusProcessing, resp.GetStatus())
}

func TestApplyGroupsAndProcess_MLEnabled_StaysProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: "DRAFT_GROUPING_REVIEW", Category: "vacation", PrivacyLevel: "private"}, nil)
	tripRepo.EXPECT().SetStatus("t1", models.TripStatusProcessing).Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	svc.SetMLEnabled(true)
	resp, err := svc.ApplyGroupsAndProcess(ctxWithUser("u1"), &pb.ApplyGroupsAndProcessRequest{TripId: "t1"})
	require.NoError(t, err)
	require.Equal(t, models.TripStatusProcessing, resp.GetStatus())
}

func TestApplyGroupsAndProcess_DeletesRejectedMedia(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: "DRAFT_GROUPING_REVIEW", PrivacyLevel: "private"}, nil)
	mediaRepo.EXPECT().GetByID("m1").Return(&models.Media{ID: "m1", TripID: "t1", S3Key: "key"}, nil)
	mediaRepo.EXPECT().DeleteByIDs([]string{"m1"}).Return(nil)
	tripRepo.EXPECT().SetStatus("t1", models.TripStatusProcessing).Return(nil)
	tripRepo.EXPECT().SetStatus("t1", models.TripStatusDraftFinalReview).Return(nil)
	eventRepo.EXPECT().DeleteTripEventStream(gomock.Any(), "t1").Return(nil)
	eventRepo.EXPECT().PublishTripEventWS(gomock.Any(), "t1", repositories.EventTripProcessingCompleted, gomock.Any()).Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ApplyGroupsAndProcess(ctxWithUser("u1"), &pb.ApplyGroupsAndProcessRequest{
		TripId: "t1", DeletedMediaIds: []string{"m1"},
	})
	require.NoError(t, err)
}

func TestApplyGroupsAndProcess_RejectsForeignMediaForDeletion(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", Status: "DRAFT_GROUPING_REVIEW"}, nil)
	mediaRepo.EXPECT().GetByID("m1").Return(&models.Media{ID: "m1", TripID: "OTHER"}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ApplyGroupsAndProcess(ctxWithUser("u1"), &pb.ApplyGroupsAndProcessRequest{
		TripId: "t1", DeletedMediaIds: []string{"m1"},
	})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}
