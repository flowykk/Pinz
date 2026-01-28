package main

import (
	"log"

	"github.com/joho/godotenv"

	"pinz/backend/auth-service/internal/db"
	"pinz/backend/auth-service/internal/di"
	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/server"
)

func main() {
	_ = godotenv.Load()

	sqlDB, err := db.InitDB()
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer sqlDB.Close()

	redisClient, err := repositories.InitRedisClient()
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	deps := di.BuildDependencies(sqlDB, redisClient)
	if err := server.RunGRPCServer(deps.AuthService); err != nil {
		log.Fatalf("server: %v", err)
	}
}
