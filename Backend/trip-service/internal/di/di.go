package di

import (
	"database/sql"

	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/services"
	pb "pinz/backend/trip-service/pkg/proto"
)

type Dependencies struct {
	TripService pb.TripServiceServer
}

func BuildDependencies(db *sql.DB) (*Dependencies, error) {
	tripRepo := repositories.NewTripRepository(db)
	participantRepo := repositories.NewTripParticipantRepository(db)
	tripSvc := services.NewTripService(tripRepo, participantRepo)
	return &Dependencies{TripService: tripSvc}, nil
}
