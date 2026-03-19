package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"pinz/backend/trip-service/internal/mocks"
	"pinz/backend/trip-service/internal/models"
	"pinz/backend/trip-service/internal/server"
	pb "pinz/backend/trip-service/pkg/proto"
)

// ctxWithUser returns a context with user_id set via the same interceptor as in production.
func ctxWithUser(userID string) context.Context {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(server.MetadataUserIDKey, userID))
	var out context.Context
	_, _ = server.AuthUnaryInterceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/pinz.TripService/CreateTrip"}, func(ctx context.Context, _ interface{}) (interface{}, error) {
		out = ctx
		return nil, nil
	})
	return out
}

func TestTripToProto(t *testing.T) {
	now := time.Unix(1000, 0)
	cases := map[string]struct {
		trip  *models.Trip
		check func(t *testing.T, out *pb.Trip)
	}{
		"basic": {
			trip: &models.Trip{
				ID: "id1", OwnerUserID: "u1", Name: "Trip", Description: "desc",
				Category: "Отпуск", Season: "Лето", Status: "Created", PrivacyLevel: "Private",
				LikesCount: 1, DislikesCount: 0, CoverURL: "", IsPublished: false, IsGenerated: false,
				CreatedAt: now, UpdatedAt: now,
			},
			check: func(t *testing.T, out *pb.Trip) {
				require.Equal(t, "id1", out.Id)
				require.Equal(t, "u1", out.OwnerUserId)
				require.Equal(t, "Trip", out.Name)
				require.Equal(t, "desc", out.Description)
				require.Equal(t, "Отпуск", out.Category)
				require.Equal(t, "Лето", out.Season)
				require.Equal(t, int32(1), out.LikesCount)
				require.Equal(t, int64(1000), out.CreatedAtUnix)
				require.Equal(t, int64(1000), out.UpdatedAtUnix)
				require.Equal(t, int64(0), out.StartDateUnix)
				require.Equal(t, int64(0), out.EndDateUnix)
			},
		},
		"with_dates": {
			trip: &models.Trip{
				ID: "id2", OwnerUserID: "u2", Name: "T2", Description: "", Category: "Другое", Season: "Зима",
				Status: "Created", PrivacyLevel: "Public", StartDate: ptrTime(time.Unix(2000, 0)), EndDate: ptrTime(time.Unix(3000, 0)),
				CreatedAt: now, UpdatedAt: now,
			},
			check: func(t *testing.T, out *pb.Trip) {
				require.Equal(t, int64(2000), out.StartDateUnix)
				require.Equal(t, int64(3000), out.EndDateUnix)
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			out := tripToProto(tc.trip)
			require.NotNil(t, out)
			tc.check(t, out)
		})
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestCreateTrip_ValidationErrors(t *testing.T) {
	// Service with nil repos: we only hit validation, no repo calls.
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	validReq := &pb.CreateTripRequest{
		Name: "Trip", Category: "Отпуск", Season: "Лето", PrivacyLevel: "Private",
	}

	cases := map[string]struct {
		ctx  context.Context
		req  *pb.CreateTripRequest
		code codes.Code
	}{
		"missing_user": {
			ctx:  context.Background(),
			req:  validReq,
			code: codes.Unauthenticated,
		},
		"empty_name": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.CreateTripRequest{Name: "", Category: "Отпуск", Season: "Лето", PrivacyLevel: "Private"},
			code: codes.InvalidArgument,
		},
		"name_too_long": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.CreateTripRequest{Name: string(make([]byte, MaxNameLength+1)), Category: "Отпуск", Season: "Лето", PrivacyLevel: "Private"},
			code: codes.InvalidArgument,
		},
		"description_too_long": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.CreateTripRequest{Name: "Trip", Description: string(make([]byte, MaxDescriptionLength+1)), Category: "Отпуск", Season: "Лето", PrivacyLevel: "Private"},
			code: codes.InvalidArgument,
		},
		"invalid_category": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.CreateTripRequest{Name: "Trip", Category: "Invalid", Season: "Лето", PrivacyLevel: "Private"},
			code: codes.InvalidArgument,
		},
		"invalid_season": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.CreateTripRequest{Name: "Trip", Category: "Отпуск", Season: "Invalid", PrivacyLevel: "Private"},
			code: codes.InvalidArgument,
		},
		"invalid_privacy": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.CreateTripRequest{Name: "Trip", Category: "Отпуск", Season: "Лето", PrivacyLevel: "Invalid"},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.CreateTrip(tc.ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestGetTrip_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx  context.Context
		req  *pb.GetTripRequest
		code codes.Code
	}{
		"missing_user": {
			ctx:  context.Background(),
			req:  &pb.GetTripRequest{TripId: "trip-1"},
			code: codes.Unauthenticated,
		},
		"empty_trip_id": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.GetTripRequest{TripId: ""},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.GetTrip(tc.ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestUpdateTrip_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx  context.Context
		req  *pb.UpdateTripRequest
		code codes.Code
	}{
		"missing_user": {
			ctx:  context.Background(),
			req:  &pb.UpdateTripRequest{TripId: "trip-1"},
			code: codes.Unauthenticated,
		},
		"empty_trip_id": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.UpdateTripRequest{TripId: ""},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.UpdateTrip(tc.ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestDeleteTrip_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx  context.Context
		req  *pb.DeleteTripRequest
		code codes.Code
	}{
		"missing_user": {
			ctx:  context.Background(),
			req:  &pb.DeleteTripRequest{TripId: "trip-1"},
			code: codes.Unauthenticated,
		},
		"empty_trip_id": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.DeleteTripRequest{TripId: ""},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.DeleteTrip(tc.ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestRemoveParticipant_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx  context.Context
		req  *pb.RemoveParticipantRequest
		code codes.Code
	}{
		"missing_user": {
			ctx:  context.Background(),
			req:  &pb.RemoveParticipantRequest{TripId: "trip-1", UserId: "u2"},
			code: codes.Unauthenticated,
		},
		"empty_trip_id": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.RemoveParticipantRequest{TripId: "", UserId: "u2"},
			code: codes.InvalidArgument,
		},
		"empty_user_id": {
			ctx:  ctxWithUser("u1"),
			req:  &pb.RemoveParticipantRequest{TripId: "trip-1", UserId: ""},
			code: codes.InvalidArgument,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.RemoveParticipant(tc.ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestListUserTrips_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx  context.Context
		req  *pb.ListUserTripsRequest
		code codes.Code
	}{
		"missing_user": {
			ctx:  context.Background(),
			req:  &pb.ListUserTripsRequest{},
			code: codes.Unauthenticated,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.ListUserTrips(tc.ctx, tc.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
		})
	}
}

func TestListFavourites_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ListFavourites(context.Background(), &pb.ListFavouritesRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
}

func TestListFavourites_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	trip := &models.Trip{
		ID: "t1", OwnerUserID: "u1", Name: "Fav Trip", Category: "Отпуск", Season: "Лето",
		Status: "READY", PrivacyLevel: "Private", CreatedAt: time.Unix(1000, 0), UpdatedAt: time.Unix(1000, 0),
	}
	favRepo.EXPECT().ListTripIDsByUserID("user-1", int32(20), int32(0)).Return([]string{"t1"}, nil)
	tripRepo.EXPECT().GetByID("t1").Return(trip, nil)

	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, favRepo)
	resp, err := svc.ListFavourites(ctxWithUser("user-1"), &pb.ListFavouritesRequest{Limit: 20, Offset: 0})
	require.NoError(t, err)
	require.Len(t, resp.GetTrips(), 1)
	require.Equal(t, "t1", resp.GetTrips()[0].GetId())
	require.Equal(t, "Fav Trip", resp.GetTrips()[0].GetName())
}

func TestListFavourites_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)
	favRepo.EXPECT().ListTripIDsByUserID("user-1", int32(20), int32(0)).Return(nil, nil)

	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, favRepo)
	resp, err := svc.ListFavourites(ctxWithUser("user-1"), &pb.ListFavouritesRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.GetTrips())
}

func TestListFavourites_SkipsSoftDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	activeTrip := &models.Trip{
		ID: "t1", OwnerUserID: "u1", Name: "Active", Category: "Отпуск", Season: "Лето",
		Status: "READY", PrivacyLevel: "Private", IsSoftDeleted: false,
		CreatedAt: time.Unix(1000, 0), UpdatedAt: time.Unix(1000, 0),
	}
	softDeletedTrip := &models.Trip{
		ID: "t2", OwnerUserID: "u1", Name: "Deleted", Category: "Отпуск", Season: "Лето",
		Status: "READY", PrivacyLevel: "Private", IsSoftDeleted: true,
		CreatedAt: time.Unix(1000, 0), UpdatedAt: time.Unix(1000, 0),
	}
	favRepo.EXPECT().ListTripIDsByUserID("user-1", int32(20), int32(0)).Return([]string{"t1", "t2"}, nil)
	tripRepo.EXPECT().GetByID("t1").Return(activeTrip, nil)
	tripRepo.EXPECT().GetByID("t2").Return(softDeletedTrip, nil)

	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, favRepo)
	resp, err := svc.ListFavourites(ctxWithUser("user-1"), &pb.ListFavouritesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetTrips(), 1)
	require.Equal(t, "t1", resp.GetTrips()[0].GetId())
}

// --- Unit tests with gomock (plan: trip-unit) ---

func TestCreateTrip_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	tripRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(t *models.Trip) error {
		t.ID = "trip-mock-id"
		return nil
	})
	participantRepo.EXPECT().Add(gomock.Any()).DoAndReturn(func(p *models.TripParticipant) error {
		require.Equal(t, "trip-mock-id", p.TripID)
		require.True(t, p.IsAdmin)
		return nil
	})

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("owner-1")
	resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
		Name: "Trip", Category: "Отпуск", Season: "Лето", PrivacyLevel: "Private",
	})
	require.NoError(t, err)
	require.Equal(t, "trip-mock-id", resp.GetTripId())
	require.Contains(t, []string{"Created", "UPLOADING"}, resp.GetStatus())
}

func TestCreateTrip_RepoCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().Create(gomock.Any()).Return(sql.ErrConnDone)

	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("u1")
	_, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
		Name: "Trip", Category: "Отпуск", Season: "Лето", PrivacyLevel: "Private",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())
}

func TestGetTrip_ParticipantSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	trip := &models.Trip{
		ID: "t1", OwnerUserID: "u1", Name: "T", Category: "Отпуск", Season: "Лето",
		Status: "READY", PrivacyLevel: "Private", CreatedAt: time.Unix(1000, 0), UpdatedAt: time.Unix(1000, 0),
	}
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID("t1").Return(trip, nil)
	participantRepo.EXPECT().IsParticipant("t1", "user-1").Return(true, nil)
	pinRepo.EXPECT().ListByTripID("t1").Return(nil, nil)
	tagRepo.EXPECT().GetByTripID("t1").Return(map[string][]string{}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, pinRepo, tagRepo, nil, favRepo)
	ctx := ctxWithUser("user-1")
	resp, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: "t1"})
	require.NoError(t, err)
	require.Equal(t, "t1", resp.GetTrip().GetId())
	require.Equal(t, "T", resp.GetTrip().GetName())
}

func TestGetTrip_NotParticipantNorFavourite(t *testing.T) {
	ctrl := gomock.NewController(t)
	trip := &models.Trip{ID: "t1", OwnerUserID: "u1", Name: "T"}
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID("t1").Return(trip, nil)
	participantRepo.EXPECT().IsParticipant("t1", "stranger").Return(false, nil)
	favRepo.EXPECT().HasFavourite("stranger", "t1").Return(false, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, favRepo)
	ctx := ctxWithUser("stranger")
	_, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: "t1"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.PermissionDenied, st.Code())
}

func TestUpdateTrip_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	trip := &models.Trip{ID: "t1", OwnerUserID: "u1", Name: "T"}
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID("t1").Return(trip, nil)
	participantRepo.EXPECT().IsParticipant("t1", "stranger").Return(false, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("stranger")
	_, err := svc.UpdateTrip(ctx, &pb.UpdateTripRequest{TripId: "t1"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.PermissionDenied, st.Code())
}

func TestDeleteTrip_NotAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	participantRepo.EXPECT().IsAdmin("t1", "participant-1").Return(false, nil)

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("participant-1")
	_, err := svc.DeleteTrip(ctx, &pb.DeleteTripRequest{TripId: "t1"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.PermissionDenied, st.Code())
}

func TestJoinTripByToken_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	inviteRepo := mocks.NewMockInvitationLinkRepositoryInterface(ctrl)
	inviteRepo.EXPECT().GetByToken("bad-token").Return(nil, sql.ErrNoRows)

	svc := NewTripService(nil, nil, inviteRepo, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("u1")
	_, err := svc.JoinTripByToken(ctx, &pb.JoinTripByTokenRequest{Token: "bad-token"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestJoinTripByToken_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	inviteRepo := mocks.NewMockInvitationLinkRepositoryInterface(ctrl)
	link := &models.InvitationLink{
		ID: "link-1", TripID: "t1", Token: "token", ExpiresAt: time.Now().Add(-time.Hour),
	}
	inviteRepo.EXPECT().GetByToken("expired-token").Return(link, nil)

	svc := NewTripService(nil, nil, inviteRepo, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("u1")
	_, err := svc.JoinTripByToken(ctx, &pb.JoinTripByTokenRequest{Token: "expired-token"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.FailedPrecondition, st.Code())
}
