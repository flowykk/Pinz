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
	"pinz/backend/trip-service/internal/repositories"
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
		trip *models.Trip
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
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			out := svc.tripToProto(context.Background(), tc.trip)
			require.NotNil(t, out)
			tc.check(t, out)
		})
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestCreateTrip_ValidationErrors(t *testing.T) {
	// Service with nil repos: we only hit validation, no repo calls.
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	validReq := &pb.CreateTripRequest{
		Name: "Trip", Category: "Отпуск", Season: "Лето",
	}

	cases := map[string]struct {
		ctx context.Context
		req *pb.CreateTripRequest
		code codes.Code
	}{
		"missing_user": {
			ctx: context.Background(),
			req: validReq,
			code: codes.Unauthenticated,
		},
		"empty_name": {
			ctx: ctxWithUser("u1"),
			req: &pb.CreateTripRequest{Name: "", Category: "Отпуск", Season: "Лето"},
			code: codes.InvalidArgument,
		},
		"name_too_long": {
			ctx: ctxWithUser("u1"),
			req: &pb.CreateTripRequest{Name: string(make([]byte, MaxNameLength+1)), Category: "Отпуск", Season: "Лето"},
			code: codes.InvalidArgument,
		},
		"description_too_long": {
			ctx: ctxWithUser("u1"),
			req: &pb.CreateTripRequest{Name: "Trip", Description: string(make([]byte, MaxDescriptionLength+1)), Category: "Отпуск", Season: "Лето"},
			code: codes.InvalidArgument,
		},
		"invalid_category": {
			ctx: ctxWithUser("u1"),
			req: &pb.CreateTripRequest{Name: "Trip", Category: "Invalid", Season: "Лето"},
			code: codes.InvalidArgument,
		},
		"invalid_season": {
			ctx: ctxWithUser("u1"),
			req: &pb.CreateTripRequest{Name: "Trip", Category: "Отпуск", Season: "Invalid"},
			code: codes.InvalidArgument,
		},
		"unsupported_content_type": {
			ctx: ctxWithUser("u1"),
			req: &pb.CreateTripRequest{
				Name: "Trip", Category: "Отпуск", Season: "Лето",
				FilesToUpload: []*pb.FileToUpload{{ClientId: "c1", ContentType: "image/gif"}},
			},
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

func TestCreateTrip_PresignedUploadURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	urlMock := mocks.NewMockMediaURLResolver(ctrl)

	tripRepo.EXPECT().Create(gomock.Any()).Do(func(trip *models.Trip) {
		trip.ID = "trip-mock-1"
	}).Return(nil)
	participantRepo.EXPECT().Add(gomock.Any()).Return(nil)
	urlMock.EXPECT().PresignedUploadURL(gomock.Any(), gomock.Any(), "image/jpeg").DoAndReturn(
		func(_ context.Context, key, ct string) (string, error) {
			require.Contains(t, key, "trip-mock-1")
			require.Contains(t, key, "client-1")
			require.Equal(t, "image/jpeg", ct)
			return "https://storage.example/presigned-put", nil
		},
	)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, urlMock, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.CreateTrip(ctxWithUser("u1"), &pb.CreateTripRequest{
		Name: "Trip",
		Category: "Отпуск",
		Season: "Лето",
		FilesToUpload: []*pb.FileToUpload{
			{ClientId: "client-1", ContentType: "image/jpeg"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "UPLOADING", resp.GetStatus())
	require.Len(t, resp.GetUploadUrls(), 1)
	require.Equal(t, "https://storage.example/presigned-put", resp.GetUploadUrls()[0].GetUrl())
	require.Equal(t, "trips/trip-mock-1/client-1.jpg", resp.GetUploadUrls()[0].GetS3Key())
}

func TestGetTrip_ValidationErrors(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx context.Context
		req *pb.GetTripRequest
		code codes.Code
	}{
		"missing_user": {
			ctx: context.Background(),
			req: &pb.GetTripRequest{TripId: "trip-1"},
			code: codes.Unauthenticated,
		},
		"empty_trip_id": {
			ctx: ctxWithUser("u1"),
			req: &pb.GetTripRequest{TripId: ""},
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
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx context.Context
		req *pb.UpdateTripRequest
		code codes.Code
	}{
		"missing_user": {
			ctx: context.Background(),
			req: &pb.UpdateTripRequest{TripId: "trip-1"},
			code: codes.Unauthenticated,
		},
		"empty_trip_id": {
			ctx: ctxWithUser("u1"),
			req: &pb.UpdateTripRequest{TripId: ""},
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
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx context.Context
		req *pb.DeleteTripRequest
		code codes.Code
	}{
		"missing_user": {
			ctx: context.Background(),
			req: &pb.DeleteTripRequest{TripId: "trip-1"},
			code: codes.Unauthenticated,
		},
		"empty_trip_id": {
			ctx: ctxWithUser("u1"),
			req: &pb.DeleteTripRequest{TripId: ""},
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
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx context.Context
		req *pb.RemoveParticipantRequest
		code codes.Code
	}{
		"missing_user": {
			ctx: context.Background(),
			req: &pb.RemoveParticipantRequest{TripId: "trip-1", UserId: "u2"},
			code: codes.Unauthenticated,
		},
		"empty_trip_id": {
			ctx: ctxWithUser("u1"),
			req: &pb.RemoveParticipantRequest{TripId: "", UserId: "u2"},
			code: codes.InvalidArgument,
		},
		"empty_user_id": {
			ctx: ctxWithUser("u1"),
			req: &pb.RemoveParticipantRequest{TripId: "trip-1", UserId: ""},
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
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	cases := map[string]struct {
		ctx context.Context
		req *pb.ListUserTripsRequest
		code codes.Code
	}{
		"missing_user": {
			ctx: context.Background(),
			req: &pb.ListUserTripsRequest{},
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
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, favRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, favRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, favRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ListFavourites(ctxWithUser("user-1"), &pb.ListFavouritesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetTrips(), 1)
	require.Equal(t, "t1", resp.GetTrips()[0].GetId())
}

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

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("owner-1")
	resp, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
		Name: "Trip", Category: "Отпуск", Season: "Лето",
	})
	require.NoError(t, err)
	require.Equal(t, "trip-mock-id", resp.GetTripId())
	require.Contains(t, []string{"Created", "UPLOADING"}, resp.GetStatus())
}

func TestCreateTrip_RepoCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	tripRepo.EXPECT().Create(gomock.Any()).Return(sql.ErrConnDone)

	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("u1")
	_, err := svc.CreateTrip(ctx, &pb.CreateTripRequest{
		Name: "Trip", Category: "Отпуск", Season: "Лето",
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
	participantRepo.EXPECT().GetByTripID("t1").Return([]*models.TripParticipant{
		{TripID: "t1", UserID: "user-1", IsAdmin: true},
	}, nil)
	tripPrivacyRepo := mocks.NewMockTripPrivacyRepositoryInterface(ctrl)
	tripPrivacyRepo.EXPECT().GetByTripID(gomock.Any(), "t1").Return([]repositories.PrivacyEntry{
		{UserID: "user-1", PrivacyLevel: "Public"},
	}, nil)
	settingsRepo := mocks.NewMockTripSettingsRepositoryInterface(ctrl)
	settingsRepo.EXPECT().GetByTripAndUsers("t1", []string{"user-1"}).Return(map[string]bool{"user-1": false}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, settingsRepo, nil, nil, nil, pinRepo, tagRepo, nil, favRepo, nil, nil, nil, tripPrivacyRepo, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("user-1")
	resp, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: "t1"})
	require.NoError(t, err)
	require.Equal(t, "t1", resp.GetTrip().GetId())
	require.Equal(t, "T", resp.GetTrip().GetName())
	require.Len(t, resp.GetParticipants(), 1)
	require.Equal(t, "user-1", resp.GetParticipants()[0].GetUserId())
	require.Equal(t, "admin", resp.GetParticipants()[0].GetRole())
	require.Equal(t, "Public", resp.GetParticipants()[0].GetPrivacyLevel())
	require.False(t, resp.GetCurrentUserSettings().GetNotificationsEnabled())
	require.Equal(t, "Public", resp.GetCurrentUserSettings().GetPrivacyLevel())
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

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, favRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("stranger")
	_, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: "t1"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.PermissionDenied, st.Code())
}

func TestGetTrip_PublicSharedAccess_Published(t *testing.T) {
	ctrl := gomock.NewController(t)
	trip := &models.Trip{
		ID: "t1", OwnerUserID: "u1", Name: "Shared", Category: "Отпуск", Season: "Лето",
		Status: "READY", PrivacyLevel: "Public", IsPublished: true,
		CreatedAt: time.Unix(1000, 0), UpdatedAt: time.Unix(1000, 0),
	}
	publicPin := &models.Pin{ID: "p-pub", TripID: "t1", Name: "Public pin", Category: "Достопримечательность", PrivacyLevel: "Public", IsPublishedInFeed: true}
	hiddenPin := &models.Pin{ID: "p-priv", TripID: "t1", Name: "Private pin", Category: "Жилье", PrivacyLevel: "Private", IsPublishedInFeed: true}
	notPublishedPin := &models.Pin{ID: "p-unpub", TripID: "t1", Name: "Not in feed", Category: "Еда и напитки", PrivacyLevel: "Public", IsPublishedInFeed: false}

	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID("t1").Return(trip, nil)
	participantRepo.EXPECT().IsParticipant("t1", "stranger").Return(false, nil)
	favRepo.EXPECT().HasFavourite("stranger", "t1").Return(false, nil)
	pinRepo.EXPECT().ListByTripID("t1").Return([]*models.Pin{publicPin, hiddenPin, notPublishedPin}, nil)
	tagRepo.EXPECT().GetByTripID("t1").Return(map[string][]string{"p-pub": {"sea"}}, nil)
	mediaRepo.EXPECT().ListByPinID("p-pub").Return([]*models.Media{
		{ID: "m1", S3Key: "s1", MediaType: "image", PrivacyLevel: "Public"},
		{ID: "m2", S3Key: "s2", MediaType: "image", PrivacyLevel: "Private"},
	}, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, pinRepo, tagRepo, nil, favRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("stranger")
	resp, err := svc.GetTrip(ctx, &pb.GetTripRequest{TripId: "t1"})
	require.NoError(t, err)
	require.Equal(t, "t1", resp.GetTrip().GetId())
	require.True(t, resp.GetTrip().GetIsPublished())
	require.Len(t, resp.GetPins(), 1, "only public pin published in feed should be returned")
	require.Equal(t, "p-pub", resp.GetPins()[0].GetId())
	require.Equal(t, []string{"sea"}, resp.GetPins()[0].GetTags())
	require.Len(t, resp.GetPins()[0].GetMedia(), 1, "only Public media should be returned")
	require.Equal(t, "m1", resp.GetPins()[0].GetMedia()[0].GetMediaId())
	require.Empty(t, resp.GetParticipants(), "shared view does not expose participants")
	require.Nil(t, resp.GetCurrentUserSettings(), "shared view does not expose per-user settings")
	require.Nil(t, resp.GetActiveAddMediaSession())
}

func TestUpdateTrip_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	trip := &models.Trip{ID: "t1", OwnerUserID: "u1", Name: "T"}
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID("t1").Return(trip, nil)
	participantRepo.EXPECT().IsParticipant("t1", "stranger").Return(false, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

	svc := NewTripService(nil, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

	svc := NewTripService(nil, nil, inviteRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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

	svc := NewTripService(nil, nil, inviteRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxWithUser("u1")
	_, err := svc.JoinTripByToken(ctx, &pb.JoinTripByTokenRequest{Token: "expired-token"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestResolveMediaDeletionsForTrip(t *testing.T) {
	const tripID = "trip-1"

	cases := map[string]struct {
		inputIDs []string
		wantErr bool
		wantCode codes.Code
		wantAllowed []string
		wantS3Keys []string
		setupMedia func(*mocks.MockMediaRepositoryInterface)
	}{
		"empty_input": {
			inputIDs: nil,
			wantErr: false,
			wantAllowed: nil,
			wantS3Keys: nil,
			setupMedia: func(_ *mocks.MockMediaRepositoryInterface) {},
		},
		"unknown_media_id": {
			inputIDs: []string{"no-such"},
			wantErr: true,
			wantCode: codes.InvalidArgument,
			setupMedia: func(m *mocks.MockMediaRepositoryInterface) {
				m.EXPECT().GetByID("no-such").Return(nil, sql.ErrNoRows)
			},
		},
		"get_media_internal_error": {
			inputIDs: []string{"m1"},
			wantErr: true,
			wantCode: codes.Internal,
			setupMedia: func(m *mocks.MockMediaRepositoryInterface) {
				m.EXPECT().GetByID("m1").Return(nil, sql.ErrConnDone)
			},
		},
		"media_from_other_trip": {
			inputIDs: []string{"m-foreign"},
			wantErr: true,
			wantCode: codes.PermissionDenied,
			setupMedia: func(m *mocks.MockMediaRepositoryInterface) {
				m.EXPECT().GetByID("m-foreign").Return(
					&models.Media{ID: "m-foreign", TripID: "other-trip", S3Key: "k1"}, nil)
			},
		},
		"with_s3_key": {
			inputIDs: []string{"m1"},
			wantErr: false,
			wantAllowed: []string{"m1"},
			wantS3Keys: []string{"s3/k1"},
			setupMedia: func(m *mocks.MockMediaRepositoryInterface) {
				m.EXPECT().GetByID("m1").Return(
					&models.Media{ID: "m1", TripID: tripID, S3Key: "s3/k1"}, nil)
			},
		},
		"empty_s3_key": {
			inputIDs: []string{"m2"},
			wantErr: false,
			wantAllowed: []string{"m2"},
			wantS3Keys: nil,
			setupMedia: func(m *mocks.MockMediaRepositoryInterface) {
				m.EXPECT().GetByID("m2").Return(
					&models.Media{ID: "m2", TripID: tripID, S3Key: ""}, nil)
			},
		},
		"two_ids": {
			inputIDs: []string{"a", "b"},
			wantErr: false,
			wantAllowed: []string{"a", "b"},
			wantS3Keys: []string{"ka", "kb"},
			setupMedia: func(m *mocks.MockMediaRepositoryInterface) {
				m.EXPECT().GetByID("a").Return(&models.Media{ID: "a", TripID: tripID, S3Key: "ka"}, nil)
				m.EXPECT().GetByID("b").Return(&models.Media{ID: "b", TripID: tripID, S3Key: "kb"}, nil)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
			tc.setupMedia(mediaRepo)
			svc := NewTripService(nil, nil, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

			allowed, keys, err := svc.resolveMediaDeletionsForTrip(tripID, tc.inputIDs)
			if tc.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.wantCode, st.Code())
				require.Nil(t, allowed)
				require.Nil(t, keys)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantAllowed, allowed)
			require.Equal(t, tc.wantS3Keys, keys)
		})
	}
}

func TestApplyGroupsAndProcess_DeletedMediaSuccess(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	trip := &models.Trip{
		ID: tripID, Status: "DRAFT_GROUPING_REVIEW", Category: "Отпуск", PrivacyLevel: "Private",
	}
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	mediaRepo.EXPECT().GetByID("m1").Return(
		&models.Media{ID: "m1", TripID: tripID, S3Key: "s3/k1"}, nil)
	mediaRepo.EXPECT().DeleteByIDs([]string{"m1"}).Return(nil)
	tripRepo.EXPECT().SetStatus(tripID, "PROCESSING").Return(nil)
	// STUB: сразу после PROCESSING трип переводится в DRAFT_FINAL_REVIEW
	// (finalizeProcessingStub, пока ML-воркер не реализован).
	tripRepo.EXPECT().SetStatus(tripID, "DRAFT_FINAL_REVIEW").Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ApplyGroupsAndProcess(ctxWithUser(userID), &pb.ApplyGroupsAndProcessRequest{
		TripId: tripID,
		DeletedMediaIds: []string{"m1"},
	})
	require.NoError(t, err)
	require.Equal(t, "PROCESSING", resp.GetStatus())
}

func TestFinalizeTrip_MediaToDeleteSuccess(t *testing.T) {
	const tripID = "trip-1"
	const userID = "user-1"
	trip := &models.Trip{ID: tripID, Status: "PROCESSING", Category: "Отпуск", PrivacyLevel: "Private"}
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)

	participantRepo.EXPECT().IsParticipant(tripID, userID).Return(true, nil)
	tripRepo.EXPECT().GetByID(tripID).Return(trip, nil)
	mediaRepo.EXPECT().GetByID("m1").Return(
		&models.Media{ID: "m1", TripID: tripID, S3Key: "key1"}, nil)
	mediaRepo.EXPECT().DeleteByIDs([]string{"m1"}).Return(nil)
	pinRepo.EXPECT().ListByTripID(tripID).Return(nil, nil)
	mediaRepo.EXPECT().ListByTripID(tripID).Return(nil, nil)
	tripRepo.EXPECT().Update(gomock.Any()).Return(nil)
	tripRepo.EXPECT().SetStatus(tripID, "READY").Return(nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, mediaRepo, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.FinalizeTrip(ctxWithUser(userID), &pb.FinalizeTripRequest{
		TripId: tripID,
		MediaToDelete: []string{"m1"},
	})
	require.NoError(t, err)
	require.Equal(t, "READY", resp.GetStatus())
}

// --- Privacy enforcement tests ---

func TestPublishTrip_RejectsNonPublicPrivacy(t *testing.T) {
	cases := map[string]struct {
		privacyLevel string
	}{
		"private": {privacyLevel: "Private"},
		"restricted": {privacyLevel: "Restricted"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
			participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)

			tripRepo.EXPECT().GetByID("trip-1").Return(&models.Trip{
				ID: "trip-1", Status: "READY", PrivacyLevel: tc.privacyLevel,
			}, nil)
			participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil)

			svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			_, err := svc.PublishTrip(ctxWithUser("u1"), &pb.PublishTripRequest{
				TripId: "trip-1", PublishWhole: true,
			})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, codes.FailedPrecondition, st.Code())
			require.Contains(t, st.Message(), "Public privacy level")
		})
	}
}

func TestRequestTripCoverUpload_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	tripRepo.EXPECT().GetByID("trip-1").Return(&models.Trip{ID: "trip-1"}, nil)
	participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil)
	urls.EXPECT().
		PresignedUploadURL(gomock.Any(), gomock.Any(), "image/jpeg").
		DoAndReturn(func(_ context.Context, key, _ string) (string, error) {
			require.Contains(t, key, "trips/trip-1/cover/")
			require.True(t, len(key) > len("trips/trip-1/cover/"))
			require.Contains(t, key, ".jpg")
			return "https://s3/put?sig=1", nil
		})

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, urls, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.RequestTripCoverUpload(ctxWithUser("u1"), &pb.RequestTripCoverUploadRequest{
		TripId: "trip-1",
		Filename: "cover.JPG",
		ContentType: "image/jpeg",
	})
	require.NoError(t, err)
	require.Equal(t, "https://s3/put?sig=1", resp.GetUploadUrl())
	require.Contains(t, resp.GetS3Key(), "trips/trip-1/cover/")
}

func TestRequestTripCoverUpload_NotParticipant(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID("trip-1").Return(&models.Trip{ID: "trip-1"}, nil)
	participantRepo.EXPECT().IsParticipant("trip-1", "stranger").Return(false, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.RequestTripCoverUpload(ctxWithUser("stranger"), &pb.RequestTripCoverUploadRequest{
		TripId: "trip-1",
		Filename: "cover.jpg",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.PermissionDenied, st.Code())
}

func TestRequestTripCoverUpload_BadExtension(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)

	tripRepo.EXPECT().GetByID("trip-1").Return(&models.Trip{ID: "trip-1"}, nil)
	participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.RequestTripCoverUpload(ctxWithUser("u1"), &pb.RequestTripCoverUploadRequest{
		TripId: "trip-1",
		Filename: "cover.gif",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestConfirmTripCoverUpload_DeletesOldAndUpdates(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	old := &models.Trip{ID: "trip-1", CoverURL: "trips/trip-1/cover/old.jpg"}
	updated := &models.Trip{ID: "trip-1", CoverURL: "trips/trip-1/cover/new.jpg"}

	gomock.InOrder(
		tripRepo.EXPECT().GetByID("trip-1").Return(old, nil),
		participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil),
		urls.EXPECT().DeleteObject(gomock.Any(), "trips/trip-1/cover/old.jpg").Return(nil),
		tripRepo.EXPECT().UpdateCoverURL("trip-1", "trips/trip-1/cover/new.jpg").Return(nil),
		tripRepo.EXPECT().GetByID("trip-1").Return(updated, nil),
	)
	urls.EXPECT().ReadURL(gomock.Any(), "trips/trip-1/cover/new.jpg").Return("https://s3/get?sig=1", nil)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, urls, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ConfirmTripCoverUpload(ctxWithUser("u1"), &pb.ConfirmTripCoverUploadRequest{
		TripId: "trip-1",
		S3Key: "trips/trip-1/cover/new.jpg",
	})
	require.NoError(t, err)
	require.Equal(t, "https://s3/get?sig=1", resp.GetTrip().GetCoverUrl())
}

func TestDeleteTripCover_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	participantRepo := mocks.NewMockTripParticipantRepositoryInterface(ctrl)
	urls := mocks.NewMockMediaURLResolver(ctrl)

	trip := &models.Trip{ID: "trip-1", CoverURL: "trips/trip-1/cover/old.jpg"}
	cleared := &models.Trip{ID: "trip-1", CoverURL: ""}

	gomock.InOrder(
		tripRepo.EXPECT().GetByID("trip-1").Return(trip, nil),
		participantRepo.EXPECT().IsParticipant("trip-1", "u1").Return(true, nil),
		urls.EXPECT().DeleteObject(gomock.Any(), "trips/trip-1/cover/old.jpg").Return(nil),
		tripRepo.EXPECT().UpdateCoverURL("trip-1", "").Return(nil),
		tripRepo.EXPECT().GetByID("trip-1").Return(cleared, nil),
	)

	svc := NewTripService(tripRepo, participantRepo, nil, nil, nil, nil, urls, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.DeleteTripCover(ctxWithUser("u1"), &pb.DeleteTripCoverRequest{TripId: "trip-1"})
	require.NoError(t, err)
	require.Equal(t, "", resp.GetTrip().GetCoverUrl())
}

func TestSearchPins_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.SearchPins(context.Background(), &pb.SearchPinsRequest{Query: "x"})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestSearchPins_EmptyQuery(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.SearchPins(ctxWithUser("u1"), &pb.SearchPinsRequest{Query: " "})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSearchPins_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	tagRepo := mocks.NewMockTagRepositoryInterface(ctrl)

	lat, lon := 55.75, 37.62
	pins := []*models.Pin{
		{
			ID: "pin-1", TripID: "trip-1", Name: "Cafe Central", Description: "best cafe",
			Category: "Food", PrivacyLevel: "Public", MediaCount: 1,
			Latitude: &lat, Longitude: &lon,
		},
	}
	pinRepo.EXPECT().SearchByUserID("u1", "cafe", int32(20), int32(0)).Return(pins, nil)
	mediaRepo.EXPECT().ListByPinID("pin-1").Return([]*models.Media{
		{ID: "m1", TripID: "trip-1", S3Key: "k", MediaType: "photo", PrivacyLevel: "Public"},
	}, nil)
	tagRepo.EXPECT().GetByPinID("pin-1").Return([]string{"cafe", "coffee"}, nil)

	svc := NewTripService(nil, nil, nil, nil, nil, mediaRepo, nil, pinRepo, tagRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.SearchPins(ctxWithUser("u1"), &pb.SearchPinsRequest{Query: "cafe"})
	require.NoError(t, err)
	require.Len(t, resp.GetPins(), 1)
	got := resp.GetPins()[0]
	require.Equal(t, "pin-1", got.GetId())
	require.Equal(t, "trip-1", got.GetTripId())
	require.Equal(t, "Cafe Central", got.GetName())
	require.Equal(t, []string{"cafe", "coffee"}, got.GetTags())
	require.Len(t, got.GetMedia(), 1)
	require.Equal(t, "m1", got.GetMedia()[0].GetMediaId())
}

func TestSearchPins_TruncatesLongQueryAndNormalizesLimits(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)

	longQuery := ""
	for i := 0; i < MaxSearchQueryLength+50; i++ {
		longQuery += "a"
	}
	expected := longQuery[:MaxSearchQueryLength]
	// limit=0 → default 20, offset=-5 → 0
	pinRepo.EXPECT().SearchByUserID("u1", expected, int32(20), int32(0)).Return(nil, nil)

	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.SearchPins(ctxWithUser("u1"), &pb.SearchPinsRequest{Query: longQuery, Limit: 0, Offset: -5})
	require.NoError(t, err)
	require.Empty(t, resp.GetPins())
}

func TestSearchPins_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	pinRepo.EXPECT().SearchByUserID("u1", "cafe", int32(20), int32(0)).Return(nil, sql.ErrConnDone)

	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, pinRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.SearchPins(ctxWithUser("u1"), &pb.SearchPinsRequest{Query: "cafe"})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestListFeed_Unauthenticated(t *testing.T) {
	svc := NewTripService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.ListFeed(context.Background(), &pb.ListFeedRequest{})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestListFeed_EmptyResultDoesNotQueryUserState(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	// no calls to socialRepo / favouriteRepo expected when 0 trips returned
	tripRepo.EXPECT().ListFeed(int32(20), int32(0), "", "", []int(nil), "date").Return(nil, nil)

	svc := NewTripService(tripRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ListFeed(ctxWithUser("u1"), &pb.ListFeedRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.GetItems())
}

func TestListFeed_PerUserFlags(t *testing.T) {
	cases := map[string]struct {
		reactions  map[string]string
		favourites map[string]struct{}
		wantLiked  bool
		wantDisl   bool
		wantSaved  bool
	}{
		"no_state":         {nil, nil, false, false, false},
		"like_only":        {map[string]string{"trip-1": "Like"}, nil, true, false, false},
		"dislike_only":     {map[string]string{"trip-1": "Dislike"}, nil, false, true, false},
		"saved_only":       {nil, map[string]struct{}{"trip-1": {}}, false, false, true},
		"like_and_saved":   {map[string]string{"trip-1": "Like"}, map[string]struct{}{"trip-1": {}}, true, false, true},
		"unknown_reaction": {map[string]string{"trip-1": "Wat"}, nil, false, false, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
			pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
			mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
			socialRepo := mocks.NewMockSocialRepositoryInterface(ctrl)
			favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)

			now := time.Unix(1000, 0)
			trips := []*models.Trip{{
				ID: "trip-1", OwnerUserID: "owner", Name: "T", Category: "Отпуск", Season: "Лето",
				Status: "READY", PrivacyLevel: "Public", IsPublished: true, CreatedAt: now, UpdatedAt: now,
			}}
			tripRepo.EXPECT().ListFeed(int32(20), int32(0), "", "", []int(nil), "date").Return(trips, nil)
			socialRepo.EXPECT().GetReactionsByUserAndTrips("u1", []string{"trip-1"}).Return(tc.reactions, nil)
			favRepo.EXPECT().FavouritesByUserAndTrips("u1", []string{"trip-1"}).Return(tc.favourites, nil)
			pinRepo.EXPECT().ListPublishedPinsByTripIDs([]string{"trip-1"}).Return(map[string][]*repositories.FeedPin{}, nil)
			mediaRepo.EXPECT().TopMediaByTripIDs([]string{"trip-1"}, 8).Return(map[string][]*repositories.FeedMedia{}, nil)
			mediaRepo.EXPECT().TopMediaByPinIDs(gomock.Any(), 10).Return(map[string][]*repositories.FeedMedia{}, nil)

			svc := NewTripService(tripRepo, nil, nil, nil, nil, mediaRepo, nil, pinRepo, nil, socialRepo, favRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			resp, err := svc.ListFeed(ctxWithUser("u1"), &pb.ListFeedRequest{})
			require.NoError(t, err)
			require.Len(t, resp.GetItems(), 1)
			require.Equal(t, tc.wantLiked, resp.GetItems()[0].GetIsLiked())
			require.Equal(t, tc.wantDisl, resp.GetItems()[0].GetIsDisliked())
			require.Equal(t, tc.wantSaved, resp.GetItems()[0].GetIsSaved())
		})
	}
}

func TestListFeed_DegradesOnUserStateRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	tripRepo := mocks.NewMockTripRepositoryInterface(ctrl)
	pinRepo := mocks.NewMockPinRepositoryInterface(ctrl)
	mediaRepo := mocks.NewMockMediaRepositoryInterface(ctrl)
	socialRepo := mocks.NewMockSocialRepositoryInterface(ctrl)
	favRepo := mocks.NewMockFavouriteRepositoryInterface(ctrl)

	now := time.Unix(1000, 0)
	trips := []*models.Trip{{
		ID: "trip-1", OwnerUserID: "owner", Name: "T", Category: "Отпуск", Season: "Лето",
		Status: "READY", PrivacyLevel: "Public", IsPublished: true, CreatedAt: now, UpdatedAt: now,
	}}
	tripRepo.EXPECT().ListFeed(int32(20), int32(0), "", "", []int(nil), "date").Return(trips, nil)
	socialRepo.EXPECT().GetReactionsByUserAndTrips("u1", []string{"trip-1"}).Return(nil, sql.ErrConnDone)
	favRepo.EXPECT().FavouritesByUserAndTrips("u1", []string{"trip-1"}).Return(nil, sql.ErrConnDone)
	pinRepo.EXPECT().ListPublishedPinsByTripIDs([]string{"trip-1"}).Return(map[string][]*repositories.FeedPin{}, nil)
	mediaRepo.EXPECT().TopMediaByTripIDs([]string{"trip-1"}, 8).Return(map[string][]*repositories.FeedMedia{}, nil)
	mediaRepo.EXPECT().TopMediaByPinIDs(gomock.Any(), 10).Return(map[string][]*repositories.FeedMedia{}, nil)

	svc := NewTripService(tripRepo, nil, nil, nil, nil, mediaRepo, nil, pinRepo, nil, socialRepo, favRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	resp, err := svc.ListFeed(ctxWithUser("u1"), &pb.ListFeedRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetItems(), 1)
	require.False(t, resp.GetItems()[0].GetIsLiked())
	require.False(t, resp.GetItems()[0].GetIsDisliked())
	require.False(t, resp.GetItems()[0].GetIsSaved())
}
