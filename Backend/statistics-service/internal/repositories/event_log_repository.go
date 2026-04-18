package repositories

import (
	"context"
	"database/sql"
)

type EventLogRepository struct {
	db *sql.DB
}

func NewEventLogRepository(db *sql.DB) *EventLogRepository {
	return &EventLogRepository{db: db}
}

func (r *EventLogRepository) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM stats_event_log WHERE event_id = $1)`, eventID).Scan(&exists)
	return exists, err
}

// MarkProcessed помечает event как обработанный. Использует ON CONFLICT DO NOTHING,
// чтобы при случайных ретраях не ломать consumer.
func (r *EventLogRepository) MarkProcessed(ctx context.Context, eventID, eventType string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO stats_event_log (event_id, event_type) VALUES ($1, $2) ON CONFLICT (event_id) DO NOTHING`,
		eventID, eventType)
	return err
}
