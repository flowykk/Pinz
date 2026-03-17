package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
)

type AddMediaSessionRepository struct {
	db *sql.DB
}

func NewAddMediaSessionRepository(db *sql.DB) *AddMediaSessionRepository {
	return &AddMediaSessionRepository{db: db}
}

func (r *AddMediaSessionRepository) Create(ctx context.Context, tripID string, existingMediaIDs []string) (sessionID string, err error) {
	b, err := json.Marshal(existingMediaIDs)
	if err != nil {
		return "", err
	}
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO add_media_sessions (trip_id, existing_media_ids) VALUES ($1, $2) RETURNING session_id`,
		tripID, b,
	).Scan(&sessionID)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (r *AddMediaSessionRepository) GetExistingMediaIDs(ctx context.Context, sessionID string) ([]string, string, error) {
	var tripID string
	var raw []byte
	if err := r.db.QueryRowContext(ctx, `SELECT trip_id, existing_media_ids FROM add_media_sessions WHERE session_id = $1`, sessionID).Scan(&tripID, &raw); err != nil {
		return nil, "", err
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, "", err
	}
	return ids, tripID, nil
}

func (r *AddMediaSessionRepository) Exists(ctx context.Context, tripID, sessionID string) (bool, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM add_media_sessions WHERE trip_id = $1 AND session_id = $2`, tripID, sessionID).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}
