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

func TestUpsertPinPrivacy_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertPinPrivacy(context.Background(), &pb.UpsertPinPrivacyRequest{TripId: "t1", PinId: "p1", PrivacyLevel: "public"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUpsertPinPrivacy_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]*pb.UpsertPinPrivacyRequest{
		"empty_trip":    {TripId: "", PinId: "p1", PrivacyLevel: "public"},
		"empty_pin":     {TripId: "t1", PinId: "", PrivacyLevel: "public"},
		"invalid_level": {TripId: "t1", PinId: "p1", PrivacyLevel: "weird"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.UpsertPinPrivacy(ctxWithUser("u1"), req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestUpsertPinPrivacy_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(false, nil)
	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertPinPrivacy(ctxWithUser("u1"), &pb.UpsertPinPrivacyRequest{TripId: "t1", PinId: "p1", PrivacyLevel: "public"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestUpsertPinPrivacy_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsGenerated: true}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertPinPrivacy(ctxWithUser("u1"), &pb.UpsertPinPrivacyRequest{TripId: "t1", PinId: "p1", PrivacyLevel: "public"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpsertPinPrivacy_PinNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	pinRepo.EXPECT().GetByID("p1").Return(nil, sql.ErrNoRows)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertPinPrivacy(ctxWithUser("u1"), &pb.UpsertPinPrivacyRequest{TripId: "t1", PinId: "p1", PrivacyLevel: "public"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpsertPinPrivacy_PinForeignTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	pinRepo.EXPECT().GetByID("p1").Return(&models.Pin{ID: "p1", TripID: "OTHER"}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertPinPrivacy(ctxWithUser("u1"), &pb.UpsertPinPrivacyRequest{TripId: "t1", PinId: "p1", PrivacyLevel: "public"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpsertPinPrivacy_PinRestrictedNotChangeable(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	pinRepo.EXPECT().GetByID("p1").Return(&models.Pin{ID: "p1", TripID: "t1", PrivacyLevel: "restricted"}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertPinPrivacy(ctxWithUser("u1"), &pb.UpsertPinPrivacyRequest{TripId: "t1", PinId: "p1", PrivacyLevel: "public"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpsertPinPrivacy_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	pinPrivacyRepo := mocks.NewMockPinPrivacyRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	pinRepo.EXPECT().GetByID("p1").Return(&models.Pin{ID: "p1", TripID: "t1", PrivacyLevel: "public"}, nil)
	pinPrivacyRepo.EXPECT().Upsert(gomock.Any(), "p1", "u1", "private").Return(nil)
	pinPrivacyRepo.EXPECT().GetByPinID(gomock.Any(), "p1").Return([]repositories.PrivacyEntry{
		{UserID: "u1", PrivacyLevel: "private"},
	}, nil)
	pinRepo.EXPECT().SetPrivacyLevel("p1", "private").Return(nil)
	eventRepo.EXPECT().PublishPrivacyEvent(gomock.Any(), "pin", "p1", "t1", "u1", "private").Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, pinPrivacyRepo, nil, nil, nil, nil)
	resp, err := svc.UpsertPinPrivacy(ctxWithUser("u1"), &pb.UpsertPinPrivacyRequest{TripId: "t1", PinId: "p1", PrivacyLevel: "private"})
	require.NoError(t, err)
	require.Equal(t, "private", resp.GetEffectivePrivacyLevel())
}

func TestUpsertMediaPrivacy_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertMediaPrivacy(context.Background(), &pb.UpsertMediaPrivacyRequest{TripId: "t1", MediaId: "m1", PrivacyLevel: "public"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUpsertMediaPrivacy_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]*pb.UpsertMediaPrivacyRequest{
		"empty_trip":    {TripId: "", MediaId: "m1", PrivacyLevel: "public"},
		"empty_media":   {TripId: "t1", MediaId: "", PrivacyLevel: "public"},
		"invalid_level": {TripId: "t1", MediaId: "m1", PrivacyLevel: "weird"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.UpsertMediaPrivacy(ctxWithUser("u1"), req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestUpsertMediaPrivacy_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsGenerated: true}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertMediaPrivacy(ctxWithUser("u1"), &pb.UpsertMediaPrivacyRequest{TripId: "t1", MediaId: "m1", PrivacyLevel: "public"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpsertMediaPrivacy_MediaForeignTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	mediaRepo.EXPECT().GetByID("m1").Return(&models.Media{ID: "m1", TripID: "OTHER"}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpsertMediaPrivacy(ctxWithUser("u1"), &pb.UpsertMediaPrivacyRequest{TripId: "t1", MediaId: "m1", PrivacyLevel: "public"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpsertMediaPrivacy_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	mediaPrivacyRepo := mocks.NewMockMediaPrivacyRepositoryInterface(ctrl)
	eventRepo := mocks.NewMockTripEventPublisher(ctrl)

	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	mediaRepo.EXPECT().GetByID("m1").Return(&models.Media{ID: "m1", TripID: "t1", PrivacyLevel: "public"}, nil)
	mediaPrivacyRepo.EXPECT().Upsert(gomock.Any(), "m1", "u1", "private").Return(nil)
	mediaPrivacyRepo.EXPECT().GetByMediaID(gomock.Any(), "m1").Return([]repositories.PrivacyEntry{
		{UserID: "u1", PrivacyLevel: "private"},
	}, nil)
	mediaRepo.EXPECT().SetPrivacyLevel("m1", "private").Return(nil)
	eventRepo.EXPECT().PublishPrivacyEvent(gomock.Any(), "media", "m1", "t1", "u1", "private").Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, eventRepo, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, mediaPrivacyRepo, nil, nil, nil)
	resp, err := svc.UpsertMediaPrivacy(ctxWithUser("u1"), &pb.UpsertMediaPrivacyRequest{TripId: "t1", MediaId: "m1", PrivacyLevel: "private"})
	require.NoError(t, err)
	require.Equal(t, "private", resp.GetEffectivePrivacyLevel())
}

func TestUpdateTripSettings_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdateTripSettings(context.Background(), &pb.UpdateTripSettingsRequest{TripId: "t1"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUpdateTripSettings_EmptyTripID(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdateTripSettings(ctxWithUser("u1"), &pb.UpdateTripSettingsRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUpdateTripSettings_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(false, nil)
	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdateTripSettings(ctxWithUser("u1"), &pb.UpdateTripSettingsRequest{TripId: "t1"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestUpdateTripSettings_GeneratedRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1", IsGenerated: true}, nil)
	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdateTripSettings(ctxWithUser("u1"), &pb.UpdateTripSettingsRequest{TripId: "t1"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUpdateTripSettings_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	settingsRepo := mocks.NewMockTripSettingsRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	settingsRepo.EXPECT().EnsureDefaultSettings("t1", "u1").Return(nil)
	settingsRepo.EXPECT().UpdateNotifications("t1", "u1", false).Return(nil)
	svc := NewTripService(tripRepo, participantRepo, nil, settingsRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.UpdateTripSettings(ctxWithUser("u1"), &pb.UpdateTripSettingsRequest{TripId: "t1", NotificationsEnabled: false})
	require.NoError(t, err)
	require.True(t, resp.GetSuccess())
}

func TestUpdateTripSettings_RepoError_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	settingsRepo := mocks.NewMockTripSettingsRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsParticipant("t1", "u1").Return(true, nil)
	tripRepo.EXPECT().GetByID("t1").Return(&models.Trip{ID: "t1"}, nil)
	settingsRepo.EXPECT().EnsureDefaultSettings("t1", "u1").Return(nil)
	settingsRepo.EXPECT().UpdateNotifications("t1", "u1", true).Return(errors.New("db down"))
	svc := NewTripService(tripRepo, participantRepo, nil, settingsRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.UpdateTripSettings(ctxWithUser("u1"), &pb.UpdateTripSettingsRequest{TripId: "t1", NotificationsEnabled: true})
	require.Equal(t, codes.Internal, status.Code(err))
}
