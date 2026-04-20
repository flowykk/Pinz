package di

import (
	"database/sql"

	"github.com/redis/go-redis/v9"

	"pinz/backend/notification-service/internal/apns"
	"pinz/backend/notification-service/internal/email"
	"pinz/backend/notification-service/internal/repositories"
	"pinz/backend/notification-service/internal/scheduler"
	"pinz/backend/notification-service/internal/services"
	"pinz/backend/notification-service/internal/worker"
)

type Dependencies struct {
	NotificationService *services.NotificationService
	TripEventsDeps      worker.TripEventsDeps
	EmailDeps           worker.EmailDeps
	SchedulerDeps       scheduler.Deps

	// Closers — ресурсы, которые надо закрыть в shutdown.
	TripClient *repositories.TripClient
}

func BuildDependencies(
	db *sql.DB,
	redisClient *redis.Client,
	apnsSender apns.Sender,
	emailSender *email.Sender,
) (*Dependencies, error) {
	tokensRepo := repositories.NewDeviceTokensRepository(db)
	notifLogRepo := repositories.NewNotificationLogRepository(db)

	tripClient, err := repositories.NewTripClient()
	if err != nil {
		return nil, err
	}

	notifSvc := services.NewNotificationService(tokensRepo)

	return &Dependencies{
		NotificationService: notifSvc,
		TripEventsDeps: worker.TripEventsDeps{
			Redis:          redisClient,
			Tokens:         tokensRepo,
			NotifLog:       notifLogRepo,
			TripClient:     tripClient,
			APNS:           apnsSender,
		},
		EmailDeps: worker.EmailDeps{
			Redis:  redisClient,
			Sender: emailSender,
		},
		SchedulerDeps: scheduler.Deps{
			Redis:      redisClient,
			Tokens:     tokensRepo,
			NotifLog:   notifLogRepo,
			TripClient: tripClient,
			APNS:       apnsSender,
		},
		TripClient: tripClient,
	}, nil
}
