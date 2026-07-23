package services

import (
	"context"
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

func TestLikeTrip_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LikeTrip(context.Background(), &pb.LikeTripRequest{TripId: "t1"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLikeTrip_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LikeTrip(ctxWithUser("u1"), &pb.LikeTripRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLikeTrip_TripNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(nil, errors.New("missing"))
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LikeTrip(ctxWithUser("u1"), &pb.LikeTripRequest{TripId: "t1"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestLikeTrip_NotPublished(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsPublished: false}, nil)
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LikeTrip(ctxWithUser("u1"), &pb.LikeTripRequest{TripId: "t1"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestLikeTrip_FromNoneEmitsLikeAdded(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	socialRepo := mocks.NewMockSocialRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsPublished: true}, nil)
	socialRepo.EXPECT().SetReaction("u1", "t1", "like").Return("", nil)
	eventRepo.EXPECT().PublishStatsEvent(gomock.Any(), "LIKE_ADDED", "t1", []string{"u1"}, gomock.Any()).Return(nil)
	svc := NewTripService(tripRepo, nil, nil, nil, eventRepo, nil, nil, nil, nil, socialRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.LikeTrip(ctxWithUser("u1"), &pb.LikeTripRequest{TripId: "t1"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestLikeTrip_FromDislikeRemovesDislikeAndAddsLike(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	socialRepo := mocks.NewMockSocialRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsPublished: true}, nil)
	socialRepo.EXPECT().SetReaction("u1", "t1", "like").Return("dislike", nil)
	eventRepo.EXPECT().PublishStatsEvent(gomock.Any(), "DISLIKE_REMOVED", "t1", []string{"u1"}, gomock.Any()).Return(nil)
	eventRepo.EXPECT().PublishStatsEvent(gomock.Any(), "LIKE_ADDED", "t1", []string{"u1"}, gomock.Any()).Return(nil)
	svc := NewTripService(tripRepo, nil, nil, nil, eventRepo, nil, nil, nil, nil, socialRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LikeTrip(ctxWithUser("u1"), &pb.LikeTripRequest{TripId: "t1"})
	require.NoError(t, err)
}

func TestLikeTrip_RepoError_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	socialRepo := mocks.NewMockSocialRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsPublished: true}, nil)
	socialRepo.EXPECT().SetReaction("u1", "t1", "like").Return("", errors.New("db down"))
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, socialRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LikeTrip(ctxWithUser("u1"), &pb.LikeTripRequest{TripId: "t1"})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestDislikeTrip_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.DislikeTrip(context.Background(), &pb.DislikeTripRequest{TripId: "t1"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestDislikeTrip_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.DislikeTrip(ctxWithUser("u1"), &pb.DislikeTripRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDislikeTrip_NotPublished(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.DislikeTrip(ctxWithUser("u1"), &pb.DislikeTripRequest{TripId: "t1"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestDislikeTrip_FromLikeRemovesLikeAndAddsDislike(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	socialRepo := mocks.NewMockSocialRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsPublished: true}, nil)
	socialRepo.EXPECT().SetReaction("u1", "t1", "dislike").Return("like", nil)
	eventRepo.EXPECT().PublishStatsEvent(gomock.Any(), "LIKE_REMOVED", "t1", []string{"u1"}, gomock.Any()).Return(nil)
	eventRepo.EXPECT().PublishStatsEvent(gomock.Any(), "DISLIKE_ADDED", "t1", []string{"u1"}, gomock.Any()).Return(nil)
	svc := NewTripService(tripRepo, nil, nil, nil, eventRepo, nil, nil, nil, nil, socialRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.DislikeTrip(ctxWithUser("u1"), &pb.DislikeTripRequest{TripId: "t1"})
	require.NoError(t, err)
}

func TestDislikeTrip_AlreadyDisliked_NoEventEmitted(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	socialRepo := mocks.NewMockSocialRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsPublished: true}, nil)
	socialRepo.EXPECT().SetReaction("u1", "t1", "dislike").Return("dislike", nil)
	// old == new: PublishStatsEvent должен НЕ вызываться
	svc := NewTripService(tripRepo, nil, nil, nil, eventRepo, nil, nil, nil, nil, socialRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.DislikeTrip(ctxWithUser("u1"), &pb.DislikeTripRequest{TripId: "t1"})
	require.NoError(t, err)
}

func TestGetNotificationSettings_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GetNotificationSettings(context.Background(), &pb.GetNotificationSettingsRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetNotificationSettings_EmptyUserIDs(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.GetNotificationSettings(context.Background(), &pb.GetNotificationSettingsRequest{TripId: "t1"})
	require.NoError(t, err)
	require.Empty(t, resp.GetNotificationsEnabled())
}

func TestGetNotificationSettings_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	settingsRepo := mocks.NewMockTripSettingsRepositoryInterface(ctrl)
	settingsRepo.EXPECT().GetByTripAndUsers("t1", []string{"u1", "u2"}).Return(map[string]bool{
		"u1": true, "u2": false,
	}, nil)
	svc := NewTripService(nil, nil, nil, settingsRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.GetNotificationSettings(context.Background(), &pb.GetNotificationSettingsRequest{
		TripId: "t1", UserIds: []string{"u1", "u2"},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]bool{"u1": true, "u2": false}, resp.GetNotificationsEnabled())
}

func TestGetNotificationSettings_RepoError_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	settingsRepo := mocks.NewMockTripSettingsRepositoryInterface(ctrl)
	settingsRepo.EXPECT().GetByTripAndUsers("t1", []string{"u1"}).Return(nil, errors.New("db down"))
	svc := NewTripService(nil, nil, nil, settingsRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GetNotificationSettings(context.Background(), &pb.GetNotificationSettingsRequest{
		TripId: "t1", UserIds: []string{"u1"},
	})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestListAnniversaryTrips_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().ListAnniversaryCandidates(int64(1700000000)).Return([]*repositories.NotificationTripCandidate{
		{TripID: "t1", Name: "Trip", Participants: []string{"u1"}, YearsElapsed: 1},
	}, nil)
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ListAnniversaryTrips(context.Background(), &pb.ListAnniversaryTripsRequest{TodayUnix: 1700000000})
	require.NoError(t, err)
	require.Len(t, resp.GetTrips(), 1)
	require.Equal(t, "t1", resp.GetTrips()[0].GetTripId())
	require.Equal(t, int32(1), resp.GetTrips()[0].GetYearsElapsed())
}

func TestListAnniversaryTrips_RepoError_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().ListAnniversaryCandidates(gomock.Any()).Return(nil, errors.New("db down"))
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ListAnniversaryTrips(context.Background(), &pb.ListAnniversaryTripsRequest{TodayUnix: 0})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestListEndedMonthAgoTrips_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().ListEndedMonthAgoCandidates(int64(1700000000)).Return([]*repositories.NotificationTripCandidate{
		{TripID: "t2"},
	}, nil)
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ListEndedMonthAgoTrips(context.Background(), &pb.ListEndedMonthAgoTripsRequest{TodayUnix: 1700000000})
	require.NoError(t, err)
	require.Len(t, resp.GetTrips(), 1)
	require.Equal(t, "t2", resp.GetTrips()[0].GetTripId())
}

func TestListEndedMonthAgoTrips_RepoError_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().ListEndedMonthAgoCandidates(gomock.Any()).Return(nil, errors.New("db down"))
	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ListEndedMonthAgoTrips(context.Background(), &pb.ListEndedMonthAgoTripsRequest{TodayUnix: 0})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestListTripParticipants_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ListTripParticipants(context.Background(), &pb.ListTripParticipantsRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestListTripParticipants_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().GetByTripID("t1").Return([]*models.TripParticipant{
		{TripID: "t1", UserID: "u1"},
		{TripID: "t1", UserID: "u2"},
	}, nil)
	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ListTripParticipants(context.Background(), &pb.ListTripParticipantsRequest{TripId: "t1"})
	require.NoError(t, err)
	require.Equal(t, []string{"u1", "u2"}, resp.GetUserIds())
}
