package di

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"

	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/services"
	pb "pinz/backend/auth-service/pkg/proto"
)

type Dependencies struct {
	AuthService pb.AuthServiceServer
}

func BuildDependencies(db *sql.DB, redisClient *redis.Client) (*Dependencies, error) {
	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}
	rpDisplayName := os.Getenv("WEBAUTHN_RP_DISPLAY_NAME")
	if rpDisplayName == "" {
		rpDisplayName = "Pinz"
	}
	rpOrigin := os.Getenv("WEBAUTHN_RP_ORIGIN")
	if rpOrigin == "" {
		rpOrigin = "http://localhost"
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpDisplayName,
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn init: %w", err)
	}

	userRepo := repositories.NewUserRepository(db)
	credRepo := repositories.NewCredentialRepository(db)
	redisRepo := repositories.NewRedisRepository(redisClient)
	v := validator.New()

	authSvc := services.NewAuthService(userRepo, credRepo, redisRepo, v, wa)
	return &Dependencies{AuthService: authSvc}, nil
}
