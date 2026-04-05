package di

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/repositories"
	s3storage "pinz/backend/trip-service/internal/s3"
	"pinz/backend/trip-service/internal/services"
	pb "pinz/backend/trip-service/pkg/proto"
)

type Dependencies struct {
	TripService pb.TripServiceServer
}

func BuildDependencies(ctx context.Context, db *sql.DB, redisClient *redis.Client) (*Dependencies, error) {
	tripRepo := repositories.NewTripRepository(db)
	participantRepo := repositories.NewTripParticipantRepository(db)
	inviteRepo := repositories.NewInvitationLinkRepository(db)
	settingsRepo := repositories.NewTripSettingsRepository(db)
	mediaRepo := repositories.NewMediaRepository(db)
	pinRepo := repositories.NewPinRepository(db)
	tagRepo := repositories.NewTagRepository(db)
	socialRepo := repositories.NewSocialRepository(db)
	favouriteRepo := repositories.NewFavouriteRepository(db)
	var eventPub repositories.TripEventPublisher
	if redisClient != nil {
		eventPub = repositories.NewRedisRepository(redisClient)
	} else {
		slog.Warn("trip-service: Redis not configured, trip events will not be published")
	}
	s3Client, err := s3storage.NewFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	var mediaURLs services.MediaURLResolver
	if s3Client != nil {
		mediaURLs = s3Client
	}
	tripSvc := services.NewTripService(tripRepo, participantRepo, inviteRepo, settingsRepo, eventPub, mediaRepo, mediaURLs, pinRepo, tagRepo, socialRepo, favouriteRepo)
	return &Dependencies{TripService: tripSvc}, nil
}
