package repositories

import (
	"context"

	"pinz/backend/notification-service/internal/models"
)

// DeviceTokensRepositoryInterface — CRUD над таблицей device_tokens.
type DeviceTokensRepositoryInterface interface {
	Upsert(ctx context.Context, userID, apnsToken string) (string, error)
	Delete(ctx context.Context, userID, apnsToken string) (int64, error)
	ListByUser(ctx context.Context, userID string) ([]models.DeviceToken, error)
	ListByUsers(ctx context.Context, userIDs []string) ([]models.DeviceToken, error)
	DeleteByToken(ctx context.Context, apnsToken string) (int64, error)
}

// NotificationLogRepositoryInterface — идемпотентность отправок.
type NotificationLogRepositoryInterface interface {
	MarkSent(ctx context.Context, eventID, apnsToken string) (bool, error)
	IsSent(ctx context.Context, eventID, apnsToken string) (bool, error)
}
