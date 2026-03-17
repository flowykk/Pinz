package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/worker"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	database, err := db.InitDB()
	if err != nil {
		slog.Error("worker: failed to init DB", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	redisClient, err := repositories.InitRedisClient()
	if err != nil {
		slog.Error("worker: failed to init redis", "error", err)
		os.Exit(1)
	}

	tripRepo := repositories.NewTripRepository(database)
	participantRepo := repositories.NewTripParticipantRepository(database)
	geoRepo := repositories.NewGeoRegistryRepository(database)
	mediaRepo := repositories.NewMediaRepository(database)
	tagRepo := repositories.NewTagRepository(database)
	pinRepo := repositories.NewPinRepository(database)
	tripPrivacyRepo := repositories.NewTripPrivacyRepository(database)
	pinPrivacyRepo := repositories.NewPinPrivacyRepository(database)
	mediaPrivacyRepo := repositories.NewMediaPrivacyRepository(database)
	var eventRepo *repositories.RedisRepository
	if redisClient != nil {
		eventRepo = repositories.NewRedisRepository(redisClient)
	}

	if err := worker.Run(ctx, redisClient, tripRepo, participantRepo, geoRepo, mediaRepo, tagRepo, pinRepo, eventRepo, tripPrivacyRepo, pinPrivacyRepo, mediaPrivacyRepo); err != nil {
		slog.Error("worker: run failed", "error", err)
		os.Exit(1)
	}
}
