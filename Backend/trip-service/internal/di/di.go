package di

import (
	"database/sql"

	"pinz/backend/trip-service/internal/services"
	pb "pinz/backend/trip-service/pkg/proto"
)

type Dependencies struct {
	TripService pb.TripServiceServer
}

func BuildDependencies(db *sql.DB) (*Dependencies, error) {
	tripSvc := services.NewTripService()
	return &Dependencies{TripService: tripSvc}, nil
}
