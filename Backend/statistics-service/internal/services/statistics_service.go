package services

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pinz/backend/statistics-service/internal/repositories"
	pb "pinz/backend/statistics-service/pkg/proto"
)

type StatisticsService struct {
	pb.UnimplementedStatisticsServiceServer

	userStats repositories.UserStatsRepositoryInterface
	tripLocations    repositories.TripLocationsRepositoryInterface
}

func NewStatisticsService(
	userStats repositories.UserStatsRepositoryInterface,
	tripLocations repositories.TripLocationsRepositoryInterface,
) *StatisticsService {
	return &StatisticsService{userStats: userStats, tripLocations: tripLocations}
}

// GetUserStats возвращает персональные счётчики пользователя (лайки/дизлайки/батлы).
// Коллективные счётчики (total_trips/pins/media) считает API Gateway через trip-service.
func (s *StatisticsService) GetUserStats(ctx context.Context, req *pb.GetUserStatsRequest) (*pb.GetUserStatsResponse, error) {
	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	st, err := s.userStats.GetByUserID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get stats")
	}
	return &pb.GetUserStatsResponse{Stats: &pb.UserStats{
		UserId:          st.UserID,
		TotalLikes:      st.TotalLikes,
		TotalDislikes:   st.TotalDislikes,
		BattlesFinished: st.BattlesFinished,
	}}, nil
}

// GetVisitedLocations агрегирует локации по списку trip_ids, переданному API Gateway.
// visit_count = COUNT(DISTINCT trip_id) для каждой локации.
func (s *StatisticsService) GetVisitedLocations(ctx context.Context, req *pb.GetVisitedLocationsRequest) (*pb.GetVisitedLocationsResponse, error) {
	userID := req.GetUserId()
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	typeFilter := req.GetType()
	if typeFilter != "" && typeFilter != "Country" && typeFilter != "City" {
		return nil, status.Error(codes.InvalidArgument, "type must be Country, City or empty")
	}
	locs, err := s.tripLocations.AggregateVisitedByTripIDs(ctx, req.GetTripIds(), typeFilter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to aggregate visited locations")
	}
	out := make([]*pb.VisitedLocation, 0, len(locs))
	for _, l := range locs {
		out = append(out, &pb.VisitedLocation{
			LocationId:      l.LocationID,
			Name:            l.Name,
			Type:            l.Type,
			ParentId:        l.ParentID,
			VisitCount:      l.VisitCount,
			LastVisitAtUnix: l.LastVisitAt.Unix(),
		})
	}
	return &pb.GetVisitedLocationsResponse{Locations: out}, nil
}
