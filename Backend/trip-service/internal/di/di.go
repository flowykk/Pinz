package di

import (
	"database/sql"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/services"
	pb "pinz/backend/trip-service/pkg/proto"
)

type Dependencies struct {
	TripService pb.TripServiceServer
}

func BuildDependencies(db *sql.DB, redisClient *redis.Client) (*Dependencies, error) {
	tripRepo := repositories.NewTripRepository(db)
	participantRepo := repositories.NewTripParticipantRepository(db)
	inviteRepo := repositories.NewInvitationLinkRepository(db)
	settingsRepo := repositories.NewTripSettingsRepository(db)
	mediaRepo := repositories.NewMediaRepository(db)
	pinRepo := repositories.NewPinRepository(db)
	tagRepo := repositories.NewTagRepository(db)
	var eventRepo *repositories.RedisRepository
	if redisClient != nil {
		eventRepo = repositories.NewRedisRepository(redisClient)
	} else {
		slog.Warn("trip-service: Redis not configured, trip events will not be published")
	}
	tripSvc := services.NewTripService(tripRepo, participantRepo, inviteRepo, settingsRepo, eventRepo, mediaRepo, pinRepo, tagRepo)
	return &Dependencies{TripService: tripSvc}, nil
}
