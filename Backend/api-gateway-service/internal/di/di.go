package di

import (
	auth "pinz/backend/api-gateway-service/internal/clients/auth"
	"pinz/backend/api-gateway-service/internal/handlers"
	"pinz/backend/api-gateway-service/internal/services"
)

type Dependencies struct {
	AuthHandler *handlers.AuthHandler
	AuthClient  *auth.Client
}

func BuildDependencies() (*Dependencies, error) {
	authClient, err := auth.NewClient()
	if err != nil {
		return nil, err
	}
	authSvc := services.NewAuthService(authClient)
	authHandler := handlers.NewAuthHandler(authSvc)
	return &Dependencies{
		AuthHandler: authHandler,
		AuthClient:  authClient,
	}, nil
}

func (d *Dependencies) Close() {
	if d.AuthClient != nil {
		_ = d.AuthClient.Close()
	}
}
