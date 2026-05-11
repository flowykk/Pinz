package di

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"

	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/s3"
	"pinz/backend/auth-service/internal/services"
	pb "pinz/backend/auth-service/pkg/proto"
)

type Dependencies struct {
	AuthService pb.AuthServiceServer
}

func BuildDependencies(db *sql.DB, redisClient *redis.Client) (*Dependencies, error) {
	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		rpID = "pinz.website"
	}
	rpDisplayName := os.Getenv("WEBAUTHN_RP_DISPLAY_NAME")
	if rpDisplayName == "" {
		rpDisplayName = "Pinz"
	}
	rpOrigin := os.Getenv("WEBAUTHN_RP_ORIGIN")
	if rpOrigin == "" {
		rpOrigin = "https://pinz.website"
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPID: rpID,
		RPDisplayName: rpDisplayName,
		RPOrigins: []string{rpOrigin},
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn init: %w", err)
	}

	userRepo := repositories.NewUserRepository(db)
	credRepo := repositories.NewCredentialRepository(db)
	redisRepo := repositories.NewRedisRepository(redisClient)
	desiredPlaceRepo := repositories.NewDesiredPlaceRepository(db)
	v := validator.New()

	s3Client, err := s3.NewFromEnv(context.Background())
	if err != nil {
		return nil, fmt.Errorf("s3 init: %w", err)
	}
	var s3Uploader services.S3Uploader
	if s3Client != nil {
		s3Uploader = s3Client
	} else {
		slog.Warn("S3 not configured — avatar upload will be unavailable")
	}

	devLoginEnabled := strings.EqualFold(strings.TrimSpace(os.Getenv("AUTH_DEV_LOGIN_ENABLED")), "true")
	if devLoginEnabled {
		slog.Warn("AUTH_DEV_LOGIN_ENABLED=true — dev login RPC is exposed; never enable in prod")
	}

	authSvc := services.NewAuthService(
		userRepo, credRepo, redisRepo, desiredPlaceRepo, v, wa, s3Uploader,
		services.WithDevLogin(devLoginEnabled),
	)
	return &Dependencies{AuthService: authSvc}, nil
}
