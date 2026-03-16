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
	WSHandler   *handlers.WSHandler
}

func BuildDependencies() (*Dependencies, error) {
	authClient, err := auth.NewClient()
	if err != nil {
		return nil, err
	}
	authSvc := services.NewAuthService(authClient)
	authHandler := handlers.NewAuthHandler(authSvc)

	redisClient, err := services.InitRedisClient()
	if err != nil {
		_ = authClient.Close()
		return nil, err
	}

	tripClient, err := trip.NewClient()
	if err != nil {
		_ = authClient.Close()
		if redisClient != nil {
			_ = redisClient.Close()
		}
		return nil, err
	}
	tripHandler := handlers.NewTripHandler(tripClient)
	wsHandler := handlers.NewWSHandler(redisClient)

	return &Dependencies{
		AuthHandler: authHandler,
		AuthClient:  authClient,
		TripHandler: tripHandler,
		TripClient:  tripClient,
		WSHandler:   wsHandler,
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
