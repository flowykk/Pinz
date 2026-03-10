// @title Pinz API Gateway
// @version 1.0
// @description API Gateway for Pinz mobile client
// @host pinz.example.com
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT access token.
package main

import (
	"log"

	"github.com/joho/godotenv"

	"pinz/backend/api-gateway-service/internal/di"
	"pinz/backend/api-gateway-service/internal/server"

	_ "pinz/backend/api-gateway-service/docs"
)

func main() {
	_ = godotenv.Load()

	deps, err := di.BuildDependencies()
	if err != nil {
		log.Fatalf("deps: %v", err)
	}
	defer deps.Close()

	srv := server.NewServer(deps)
	if err := srv.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}
}
