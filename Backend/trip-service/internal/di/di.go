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

	// Worker dependencies — used by worker.Run goroutine in main.
	RedisClient         *redis.Client
	TripRepo            *repositories.TripRepository
	ParticipantRepo     *repositories.TripParticipantRepository
	GeoRepo             *repositories.GeoRegistryRepository
	MediaRepo           *repositories.MediaRepository
	TagRepo             *repositories.TagRepository
	PinRepo             *repositories.PinRepository
	EventRepo           *repositories.RedisRepository
	TripPrivacyRepo     *repositories.TripPrivacyRepository
	PinPrivacyRepo      *repositories.PinPrivacyRepository
	MediaPrivacyRepo    *repositories.MediaPrivacyRepository
	AddMediaSessionRepo *repositories.AddMediaSessionRepository
	BattleRepo          *repositories.MediaBattleRepository
	Geocoder            services.LocationResolver
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
	geoRepo := repositories.NewGeoRegistryRepository(db)
	geocoder := services.NewGeocodingClientFromEnv()

	var eventPub repositories.TripEventPublisher
	var eventRepo *repositories.RedisRepository
	if redisClient != nil {
		eventRepo = repositories.NewRedisRepository(redisClient)
		eventPub = eventRepo
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
	} else {
		slog.Info("trip-service: S3 not configured (S3_BUCKET unset); presigned upload URLs will be empty")
	}

	tripPrivacyRepo := repositories.NewTripPrivacyRepository(db)
	pinPrivacyRepo := repositories.NewPinPrivacyRepository(db)
	mediaPrivacyRepo := repositories.NewMediaPrivacyRepository(db)
	addMediaSessionRepo := repositories.NewAddMediaSessionRepository(db)
	battleRepo := repositories.NewMediaBattleRepository(db)

	tripSvc := services.NewTripService(tripRepo, participantRepo, inviteRepo, settingsRepo, eventPub, mediaRepo, mediaURLs, pinRepo, tagRepo, socialRepo, favouriteRepo, geocoder, geoRepo, addMediaSessionRepo, battleRepo)
	return &Dependencies{
		TripService:         tripSvc,
		RedisClient:         redisClient,
		TripRepo:            tripRepo,
		ParticipantRepo:     participantRepo,
		GeoRepo:             geoRepo,
		MediaRepo:           mediaRepo,
		TagRepo:             tagRepo,
		PinRepo:             pinRepo,
		EventRepo:           eventRepo,
		TripPrivacyRepo:     tripPrivacyRepo,
		PinPrivacyRepo:      pinPrivacyRepo,
		MediaPrivacyRepo:    mediaPrivacyRepo,
		AddMediaSessionRepo: addMediaSessionRepo,
		BattleRepo:          battleRepo,
		Geocoder:            geocoder,
	}, nil
}
