package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	pb "pinz/backend/trip-service/pkg/proto"
)

func TestGenerateInviteLink_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GenerateInviteLink(context.Background(), &pb.GenerateInviteLinkRequest{TripId: "t1"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGenerateInviteLink_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GenerateInviteLink(ctxWithUser("u1"), &pb.GenerateInviteLinkRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGenerateInviteLink_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(false, nil)
	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GenerateInviteLink(ctxWithUser("u1"), &pb.GenerateInviteLinkRequest{TripId: "t1"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGenerateInviteLink_TripNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(nil, sql.ErrNoRows)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GenerateInviteLink(ctxWithUser("u1"), &pb.GenerateInviteLinkRequest{TripId: "t1"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestGenerateInviteLink_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsGenerated: true}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GenerateInviteLink(ctxWithUser("u1"), &pb.GenerateInviteLinkRequest{TripId: "t1"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGenerateInviteLink_HappyPath_AppliesDefaultExpiry(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	inviteRepo := mocks.NewMockInvitationLinkRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	inviteRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(link *models.InvitationLink) error {
		require.Equal(t, "t1", link.TripID)
		require.NotEmpty(t, link.Token)
		require.NotEmpty(t, link.ID)
		require.WithinDuration(t, time.Now().Add(defaultInviteExpiresInSec*time.Second), link.ExpiresAt, 5*time.Second)
		return nil
	})

	svc := NewTripService(tripRepo, participantRepo, inviteRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.GenerateInviteLink(ctxWithUser("u1"), &pb.GenerateInviteLinkRequest{TripId: "t1"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetToken())
	require.NotEmpty(t, resp.GetInviteLinkId())
	require.Greater(t, resp.GetExpiresAtUnix(), time.Now().Unix())
}

func TestGenerateInviteLink_CapsExpiryAt30Days(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	inviteRepo := mocks.NewMockInvitationLinkRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	inviteRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(link *models.InvitationLink) error {
		require.WithinDuration(t, time.Now().Add(30*24*time.Hour), link.ExpiresAt, 5*time.Second)
		return nil
	})

	svc := NewTripService(tripRepo, participantRepo, inviteRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GenerateInviteLink(ctxWithUser("u1"), &pb.GenerateInviteLinkRequest{TripId: "t1", ExpiresInSeconds: 99999999})
	require.NoError(t, err)
}

func TestJoinTripByToken_AlreadyJoined(t *testing.T) {
	ctrl := gomock.NewController(t)
	inviteRepo := mocks.NewMockInvitationLinkRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	inviteRepo.EXPECT().GetByToken("tok").Return(&models.InvitationLink{
		TripID: "t1", Token: "tok", ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	svc := NewTripService(nil, participantRepo, inviteRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.JoinTripByToken(ctxWithUser("u1"), &pb.JoinTripByTokenRequest{Token: "tok"})
	require.NoError(t, err)
	require.True(t, resp.GetAlreadyJoined())
	require.Equal(t, "t1", resp.GetTripId())
}

func TestJoinTripByToken_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	inviteRepo := mocks.NewMockInvitationLinkRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	inviteRepo.EXPECT().GetByToken("tok").Return(&models.InvitationLink{TripID: "t1", ExpiresAt: time.Now().Add(time.Hour)}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(false, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsGenerated: true}, nil)
	svc := NewTripService(tripRepo, participantRepo, inviteRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.JoinTripByToken(ctxWithUser("u1"), &pb.JoinTripByTokenRequest{Token: "tok"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestRemoveParticipant_SelfRemoveRejected(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.RemoveParticipant(ctxWithUser("u1"), &pb.RemoveParticipantRequest{TripId: "t1", UserId: "u1"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRemoveParticipant_NotAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(false, nil)
	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.RemoveParticipant(ctxWithUser("u1"), &pb.RemoveParticipantRequest{TripId: "t1", UserId: "u2"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestRemoveParticipant_TargetNotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u2").Return(false, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.RemoveParticipant(ctxWithUser("u1"), &pb.RemoveParticipantRequest{TripId: "t1", UserId: "u2"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestRemoveParticipant_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsGenerated: true}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.RemoveParticipant(ctxWithUser("u1"), &pb.RemoveParticipantRequest{TripId: "t1", UserId: "u2"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestRemoveParticipant_HappyPath_PublishesEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u2").Return(true, nil)
	participantRepo.EXPECT().Remove("t1", "u2").Return(nil)
	eventRepo.EXPECT().PublishTripEvent(gomock.Any(), "PARTICIPANT_REMOVED", "t1", "u2").Return(nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.RemoveParticipant(ctxWithUser("u1"), &pb.RemoveParticipantRequest{TripId: "t1", UserId: "u2"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestLeaveTrip_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LeaveTrip(context.Background(), &pb.LeaveTripRequest{TripId: "t1"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLeaveTrip_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LeaveTrip(ctxWithUser("u1"), &pb.LeaveTripRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLeaveTrip_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(false, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(false, nil)
	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LeaveTrip(ctxWithUser("u1"), &pb.LeaveTripRequest{TripId: "t1"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestLeaveTrip_LastParticipant_DeletesTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	participantRepo.EXPECT().Remove("t1", "u1").Return(nil)
	participantRepo.EXPECT().GetByTripID("t1").Return([]*models.TripParticipant{}, nil)
	tripRepo.EXPECT().Delete("t1").Return(nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.LeaveTrip(ctxWithUser("u1"), &pb.LeaveTripRequest{TripId: "t1"})
	require.NoError(t, err)
	require.True(t, resp.GetTripDeleted())
}

func TestLeaveTrip_AdminLeaves_ReassignsAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	participantRepo.EXPECT().Remove("t1", "u1").Return(nil)
	eventRepo.EXPECT().PublishTripEvent(gomock.Any(), "PARTICIPANT_LEFT", "t1", "u1").Return(nil)
	participantRepo.EXPECT().GetByTripID("t1").Return([]*models.TripParticipant{
		{TripID: "t1", UserID: "u2"},
	}, nil)
	participantRepo.EXPECT().SetAdmin("t1", "u2").Return(nil)
	eventRepo.EXPECT().PublishTripEvent(gomock.Any(), "ADMIN_CHANGED", "t1", "u2").Return(nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.LeaveTrip(ctxWithUser("u1"), &pb.LeaveTripRequest{TripId: "t1"})
	require.NoError(t, err)
	require.False(t, resp.GetTripDeleted())
}

func TestLeaveTrip_NonAdmin_NoReassign(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u2").Return(false, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u2").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	participantRepo.EXPECT().Remove("t1", "u2").Return(nil)
	participantRepo.EXPECT().GetByTripID("t1").Return([]*models.TripParticipant{
		{TripID: "t1", UserID: "u1", IsAdmin: true},
	}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.LeaveTrip(ctxWithUser("u2"), &pb.LeaveTripRequest{TripId: "t1"})
	require.NoError(t, err)
	require.False(t, resp.GetTripDeleted())
}

func TestLeaveTrip_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsGenerated: true}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LeaveTrip(ctxWithUser("u1"), &pb.LeaveTripRequest{TripId: "t1"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestTransferAdmin_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.TransferAdmin(context.Background(), &pb.TransferAdminRequest{TripId: "t1", NewAdminUserId: "u2"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestTransferAdmin_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]*pb.TransferAdminRequest{
		"empty_trip":  {TripId: "", NewAdminUserId: "u2"},
		"empty_admin": {TripId: "t1", NewAdminUserId: ""},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.TransferAdmin(ctxWithUser("u1"), req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestTransferAdmin_NotAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(false, nil)
	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.TransferAdmin(ctxWithUser("u1"), &pb.TransferAdminRequest{TripId: "t1", NewAdminUserId: "u2"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestTransferAdmin_TargetNotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u2").Return(false, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.TransferAdmin(ctxWithUser("u1"), &pb.TransferAdminRequest{TripId: "t1", NewAdminUserId: "u2"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestTransferAdmin_SelfNoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.TransferAdmin(ctxWithUser("u1"), &pb.TransferAdminRequest{TripId: "t1", NewAdminUserId: "u1"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestTransferAdmin_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	participantRepo.EXPECT().IsParticipant("t1", "u2").Return(true, nil)
	participantRepo.EXPECT().SetAdmin("t1", "u2").Return(nil)
	eventRepo.EXPECT().PublishTripEvent(gomock.Any(), "ADMIN_CHANGED", "t1", "u2").Return(nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.TransferAdmin(ctxWithUser("u1"), &pb.TransferAdminRequest{TripId: "t1", NewAdminUserId: "u2"})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestTransferAdmin_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsGenerated: true}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.TransferAdmin(ctxWithUser("u1"), &pb.TransferAdminRequest{TripId: "t1", NewAdminUserId: "u2"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}
