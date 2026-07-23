package di

import (
	"os"

	auth "pinz/backend/api-gateway-service/internal/clients/auth"
	notification "pinz/backend/api-gateway-service/internal/clients/notification"
	statistics "pinz/backend/api-gateway-service/internal/clients/statistics"
	trip "pinz/backend/api-gateway-service/internal/clients/trip"
	"pinz/backend/api-gateway-service/internal/handlers"
	"pinz/backend/api-gateway-service/internal/services"
)

// defaultTripShareLinkBase — production-домен Pinz; используется, если env
// TRIP_SHARE_LINK_BASE не задана. Поведение совпадает с iOS universal-links.
const defaultTripShareLinkBase = "https://pinz.website/trips"

type Dependencies struct {
	AuthHandler *handlers.AuthHandler
	AuthClient *auth.Client
	TripHandler *handlers.TripHandler
	TripClient *trip.Client
	StatisticsHandler *handlers.StatisticsHandler
	StatisticsClient *statistics.Client
	NotificationHandler *handlers.NotificationHandler
	NotificationClient *notification.Client
	WSHandler *handlers.WSHandler
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
		// Redis optional — gateway WebSocket handler сам проверяет на nil.
		redisClient = nil
	}

	tripClient, err := trip.NewClient()
	if err != nil {
		_ = authClient.Close()
		if redisClient != nil {
			_ = redisClient.Close()
		}
		return nil, err
	}
	shareLinkBase := os.Getenv("TRIP_SHARE_LINK_BASE")
	if shareLinkBase == "" {
		shareLinkBase = defaultTripShareLinkBase
	}
	tripHandler := handlers.NewTripHandler(tripClient, authClient, shareLinkBase)
	wsHandler := handlers.NewWSHandler(redisClient, tripClient)

	statsClient, err := statistics.NewClient()
	if err != nil {
		_ = authClient.Close()
		_ = tripClient.Close()
		if redisClient != nil {
			_ = redisClient.Close()
		}
		return nil, err
	}
	statsHandler := handlers.NewStatisticsHandler(statsClient, tripClient)

	notifClient, err := notification.NewClient()
	if err != nil {
		_ = authClient.Close()
		_ = tripClient.Close()
		_ = statsClient.Close()
		if redisClient != nil {
			_ = redisClient.Close()
		}
		return nil, err
	}
	notifHandler := handlers.NewNotificationHandler(notifClient)

	return &Dependencies{
		AuthHandler: authHandler,
		AuthClient: authClient,
		TripHandler: tripHandler,
		TripClient: tripClient,
		StatisticsHandler: statsHandler,
		StatisticsClient: statsClient,
		NotificationHandler: notifHandler,
		NotificationClient: notifClient,
		WSHandler: wsHandler,
	}, nil
}

func (d *Dependencies) Close() {
	if d.AuthClient != nil {
		_ = d.AuthClient.Close()
	}
	if d.TripClient != nil {
		_ = d.TripClient.Close()
	}
	if d.StatisticsClient != nil {
		_ = d.StatisticsClient.Close()
	}
	if d.NotificationClient != nil {
		_ = d.NotificationClient.Close()
	}
}
