package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/statistics-service/internal/mocks"
	"pinz/backend/statistics-service/internal/models"
	pb "pinz/backend/statistics-service/pkg/proto"
)

func TestGetUserStats_ValidatesUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := NewStatisticsService(mocks.NewMockUserStatsRepositoryInterface(ctrl), mocks.NewMockTripLocationsRepositoryInterface(ctrl))

	_, err := svc.GetUserStats(context.Background(), &pb.GetUserStatsRequest{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetUserStats_ReturnsZeroForMissingUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	statsRepo := mocks.NewMockUserStatsRepositoryInterface(ctrl)
	tripLocations := mocks.NewMockTripLocationsRepositoryInterface(ctrl)
	statsRepo.EXPECT().GetByUserID(gomock.Any(), "user-1").Return(&models.UserStats{UserID: "user-1"}, nil)

	svc := NewStatisticsService(statsRepo, tripLocations)
	resp, err := svc.GetUserStats(context.Background(), &pb.GetUserStatsRequest{UserId: "user-1"})
	require.NoError(t, err)
	require.Equal(t, "user-1", resp.GetStats().GetUserId())
	require.Zero(t, resp.GetStats().GetTotalLikes())
	require.Zero(t, resp.GetStats().GetTotalDislikes())
}

func TestGetUserStats_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	statsRepo := mocks.NewMockUserStatsRepositoryInterface(ctrl)
	tripLocations := mocks.NewMockTripLocationsRepositoryInterface(ctrl)
	statsRepo.EXPECT().GetByUserID(gomock.Any(), "u").Return(nil, errors.New("boom"))

	svc := NewStatisticsService(statsRepo, tripLocations)
	_, err := svc.GetUserStats(context.Background(), &pb.GetUserStatsRequest{UserId: "u"})
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestGetVisitedLocations_TypeFilterValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := NewStatisticsService(mocks.NewMockUserStatsRepositoryInterface(ctrl), mocks.NewMockTripLocationsRepositoryInterface(ctrl))

	_, err := svc.GetVisitedLocations(context.Background(), &pb.GetVisitedLocationsRequest{UserId: "u", Type: "Region"})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetVisitedLocations_Ok(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	statsRepo := mocks.NewMockUserStatsRepositoryInterface(ctrl)
	tripLocations := mocks.NewMockTripLocationsRepositoryInterface(ctrl)

	now := time.Unix(1_700_000_000, 0)
	tripLocations.EXPECT().
		AggregateVisitedByTripIDs(gomock.Any(), []string{"trip-1", "trip-2"}, "country").
		Return([]*models.VisitedLocation{
			{LocationID: 1, Name: "France", Type: "country", ParentID: 0, VisitCount: 2, LastVisitAt: now},
		}, nil)

	svc := NewStatisticsService(statsRepo, tripLocations)
	resp, err := svc.GetVisitedLocations(context.Background(), &pb.GetVisitedLocationsRequest{
		UserId: "u",
		TripIds: []string{"trip-1", "trip-2"},
		Type: "country",
	})
	require.NoError(t, err)
	require.Len(t, resp.GetLocations(), 1)
	got := resp.GetLocations()[0]
	require.Equal(t, int32(1), got.GetLocationId())
	require.Equal(t, "France", got.GetName())
	require.EqualValues(t, 2, got.GetVisitCount())
	require.EqualValues(t, now.Unix(), got.GetLastVisitAtUnix())
}

func TestGetVisitedLocations_ValidatesUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	svc := NewStatisticsService(mocks.NewMockUserStatsRepositoryInterface(ctrl), mocks.NewMockTripLocationsRepositoryInterface(ctrl))
	_, err := svc.GetVisitedLocations(context.Background(), &pb.GetVisitedLocationsRequest{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetVisitedLocations_EmptyTripIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	statsRepo := mocks.NewMockUserStatsRepositoryInterface(ctrl)
	tripLocations := mocks.NewMockTripLocationsRepositoryInterface(ctrl)
	tripLocations.EXPECT().
		AggregateVisitedByTripIDs(gomock.Any(), []string(nil), "").
		Return([]*models.VisitedLocation{}, nil)

	svc := NewStatisticsService(statsRepo, tripLocations)
	resp, err := svc.GetVisitedLocations(context.Background(), &pb.GetVisitedLocationsRequest{UserId: "u"})
	require.NoError(t, err)
	require.Empty(t, resp.GetLocations())
}
