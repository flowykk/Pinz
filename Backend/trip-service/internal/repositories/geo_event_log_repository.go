package repositories

import (
	"context"
	"database/sql"
)

// GeoEventLogRepository — лог обработанных событий из pinz:trip:geo_events.
// Используется geo consumer'ом для идемпотентности (по message id Redis Streams).
type GeoEventLogRepository struct {
	db *sql.DB
}

func NewGeoEventLogRepository(db *sql.DB) *GeoEventLogRepository {
	return &GeoEventLogRepository{db: db}
}

func (r *GeoEventLogRepository) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM geo_event_log WHERE event_id = $1)`,
		eventID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *GeoEventLogRepository) MarkProcessed(ctx context.Context, eventID, eventType string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO geo_event_log (event_id, event_type, processed_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (event_id) DO NOTHING`,
		eventID, eventType)
	return err
}
