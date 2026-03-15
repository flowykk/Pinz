package services

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "pinz/backend/trip-service/pkg/proto"
)

type TripService struct {
	pb.UnimplementedTripServiceServer
}

func NewTripService() *TripService {
	return &TripService{}
}

func (s *TripService) CreateTrip(ctx context.Context, req *pb.CreateTripRequest) (*pb.CreateTripResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "CreateTrip will be implemented in task 2")
}

func (s *TripService) GetTrip(ctx context.Context, req *pb.GetTripRequest) (*pb.GetTripResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetTrip will be implemented in task 2")
}

func (s *TripService) ListUserTrips(ctx context.Context, req *pb.ListUserTripsRequest) (*pb.ListUserTripsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ListUserTrips will be implemented in task 2")
}

func (s *TripService) UpdateTrip(ctx context.Context, req *pb.UpdateTripRequest) (*pb.UpdateTripResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "UpdateTrip will be implemented in task 2")
}

func (s *TripService) DeleteTrip(ctx context.Context, req *pb.DeleteTripRequest) (*pb.DeleteTripResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "DeleteTrip will be implemented in task 2")
}

func (s *TripService) ProcessMediaGrouping(ctx context.Context, req *pb.ProcessMediaGroupingRequest) (*pb.ProcessMediaGroupingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ProcessMediaGrouping will be implemented in phase 3")
}

func (s *TripService) ApplyGroupsAndProcess(ctx context.Context, req *pb.ApplyGroupsAndProcessRequest) (*pb.ApplyGroupsAndProcessResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "ApplyGroupsAndProcess will be implemented in phase 3")
}

func (s *TripService) GetTripReview(ctx context.Context, req *pb.GetTripReviewRequest) (*pb.GetTripReviewResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetTripReview will be implemented in phase 3")
}

func (s *TripService) FinalizeTrip(ctx context.Context, req *pb.FinalizeTripRequest) (*pb.FinalizeTripResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "FinalizeTrip will be implemented in phase 3")
}
