package di

import (
	"database/sql"
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"

	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/services"
	pb "pinz/backend/auth-service/pkg/proto"
)

type Dependencies struct {
	AuthService pb.AuthServiceServer
}

func BuildDependencies(db *sql.DB, redisClient *redis.Client) *Dependencies {
	userRepo := repositories.NewUserRepository(db)
	redisRepo := repositories.NewRedisRepository(redisClient)
	v := validator.New()
	_ = v.RegisterValidation("securepwd", func(fl validator.FieldLevel) bool {
		pwd := fl.Field().String()
		if len(pwd) < 6 {
			return false
		}
		if !regexp.MustCompile(`[A-Z]`).MatchString(pwd) {
			return false
		}
		if !regexp.MustCompile(`[a-z]`).MatchString(pwd) {
			return false
		}
		if !regexp.MustCompile(`\d`).MatchString(pwd) {
			return false
		}
		return true
	})

	authSvc := services.NewAuthService(userRepo, redisRepo, v)
	return &Dependencies{AuthService: authSvc}
}
