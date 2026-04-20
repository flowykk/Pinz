package repositories

import (
	"context"
	"database/sql"

	"pinz/backend/notification-service/internal/db/sqlcdb"
)

type NotificationLogRepository struct {
	q *sqlcdb.Queries
}

func NewNotificationLogRepository(db *sql.DB) *NotificationLogRepository {
	return &NotificationLogRepository{q: sqlcdb.New(db)}
}

// MarkSent пытается вставить запись. Возвращает true, если вставка прошла
// (первая отправка), и false если дубль — значит push уже был отправлен.
func (r *NotificationLogRepository) MarkSent(ctx context.Context, eventID, apnsToken string) (bool, error) {
	n, err := r.q.NotificationLogInsert(ctx, sqlcdb.NotificationLogInsertParams{EventID: eventID, ApnsToken: apnsToken})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *NotificationLogRepository) IsSent(ctx context.Context, eventID, apnsToken string) (bool, error) {
	ok, err := r.q.NotificationLogExists(ctx, sqlcdb.NotificationLogExistsParams{EventID: eventID, ApnsToken: apnsToken})
	if err != nil {
		return false, err
	}
	return ok, nil
}
