package di

import (
	auth "pinz/backend/api-gateway-service/internal/clients/auth"
	trip "pinz/backend/api-gateway-service/internal/clients/trip"
	"pinz/backend/api-gateway-service/internal/handlers"
	"pinz/backend/api-gateway-service/internal/services"
)

type Dependencies struct {
	AuthHandler *handlers.AuthHandler
	AuthClient  *auth.Client
	TripHandler *handlers.TripHandler
	TripClient  *trip.Client
}

func BuildDependencies() (*Dependencies, error) {
	authClient, err := auth.NewClient()
	if err != nil {
		return nil, err
	}
	authSvc := services.NewAuthService(authClient)
	authHandler := handlers.NewAuthHandler(authSvc)

	tripClient, err := trip.NewClient()
	if err != nil {
		_ = authClient.Close()
		return nil, err
	}
	tripHandler := handlers.NewTripHandler(tripClient)

	return &Dependencies{
		AuthHandler: authHandler,
		AuthClient:  authClient,
		TripHandler: tripHandler,
		TripClient:  tripClient,
	}, nil
}

func (d *Dependencies) Close() {
	if d.AuthClient != nil {
		_ = d.AuthClient.Close()
	}
	if d.TripClient != nil {
		_ = d.TripClient.Close()
	}
}
