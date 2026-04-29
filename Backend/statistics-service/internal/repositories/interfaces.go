package repositories

import (
	"context"

	"pinz/backend/statistics-service/internal/models"
)

type UserStatsRepositoryInterface interface {
	GetByUserID(ctx context.Context, userID string) (*models.UserStats, error)
	IncrementLikes(ctx context.Context, userID string, delta int32) error
	IncrementDislikes(ctx context.Context, userID string, delta int32) error
	IncrementBattles(ctx context.Context, userID string, delta int32) error
}

type GeoRegistryRepositoryInterface interface {
	Upsert(ctx context.Context, loc *models.GeoLocation) error
	GetByID(ctx context.Context, id int32) (*models.GeoLocation, error)
	EnsureByName(ctx context.Context, countryName, cityName string) (*models.GeoLocation, *models.GeoLocation, error)
}

type TripLocationsRepositoryInterface interface {
	Upsert(ctx context.Context, tripID string, locationID int32) error
	DeleteByTripID(ctx context.Context, tripID string) error
	AggregateVisitedByTripIDs(ctx context.Context, tripIDs []string, typeFilter string) ([]*models.VisitedLocation, error)
}

type EventLogRepositoryInterface interface {
	IsProcessed(ctx context.Context, eventID string) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string) error
}
