package di

import (
	"database/sql"

	"github.com/redis/go-redis/v9"

	"pinz/backend/statistics-service/internal/repositories"
	"pinz/backend/statistics-service/internal/services"
	"pinz/backend/statistics-service/internal/worker"
)

type Dependencies struct {
	StatisticsService *services.StatisticsService

	WorkerDeps worker.Deps
}

func BuildDependencies(db *sql.DB, redisClient *redis.Client) (*Dependencies, error) {
	userStats := repositories.NewUserStatsRepository(db)
	geoRegistry := repositories.NewGeoRegistryRepository(db)
	tripLocations := repositories.NewTripLocationsRepository(db)
	eventLog := repositories.NewEventLogRepository(db)

	statsSvc := services.NewStatisticsService(userStats, tripLocations)

	return &Dependencies{
		StatisticsService: statsSvc,
		WorkerDeps: worker.Deps{
			Redis: redisClient,
			UserStats: userStats,
			GeoRegistry: geoRegistry,
			TripLocations: tripLocations,
			EventLog: eventLog,
		},
	}, nil
}
