package repositories

import (
	"context"
	"database/sql"
)

// PrivacyEntry is one per-user privacy choice.
type PrivacyEntry struct {
	UserID       string
	PrivacyLevel string
}

// TripPrivacyRepository handles trip_privacy table (per-user trip privacy).
type TripPrivacyRepository struct{ db *sql.DB }

func NewTripPrivacyRepository(db *sql.DB) *TripPrivacyRepository {
	return &TripPrivacyRepository{db: db}
}

func (r *TripPrivacyRepository) Upsert(ctx context.Context, tripID, userID, privacyLevel string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO trip_privacy (trip_id, user_id, privacy_level) VALUES ($1, $2, $3)
		 ON CONFLICT (trip_id, user_id) DO UPDATE SET privacy_level = $3`,
		tripID, userID, privacyLevel)
	return err
}

func (r *TripPrivacyRepository) GetByTripID(ctx context.Context, tripID string) ([]PrivacyEntry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, privacy_level FROM trip_privacy WHERE trip_id = $1`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PrivacyEntry
	for rows.Next() {
		var e PrivacyEntry
		if err := rows.Scan(&e.UserID, &e.PrivacyLevel); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// PinPrivacyRepository handles pin_privacy table.
type PinPrivacyRepository struct{ db *sql.DB }

func NewPinPrivacyRepository(db *sql.DB) *PinPrivacyRepository {
	return &PinPrivacyRepository{db: db}
}

func (r *PinPrivacyRepository) Upsert(ctx context.Context, pinID, userID, privacyLevel string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO pin_privacy (pin_id, user_id, privacy_level) VALUES ($1, $2, $3)
		 ON CONFLICT (pin_id, user_id) DO UPDATE SET privacy_level = $3`,
		pinID, userID, privacyLevel)
	return err
}

func (r *PinPrivacyRepository) GetByPinID(ctx context.Context, pinID string) ([]PrivacyEntry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, privacy_level FROM pin_privacy WHERE pin_id = $1`, pinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PrivacyEntry
	for rows.Next() {
		var e PrivacyEntry
		if err := rows.Scan(&e.UserID, &e.PrivacyLevel); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// MediaPrivacyRepository handles media_privacy table.
type MediaPrivacyRepository struct{ db *sql.DB }

func NewMediaPrivacyRepository(db *sql.DB) *MediaPrivacyRepository {
	return &MediaPrivacyRepository{db: db}
}

func (r *MediaPrivacyRepository) Upsert(ctx context.Context, mediaID, userID, privacyLevel string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO media_privacy (media_id, user_id, privacy_level) VALUES ($1, $2, $3)
		 ON CONFLICT (media_id, user_id) DO UPDATE SET privacy_level = $3`,
		mediaID, userID, privacyLevel)
	return err
}

func (r *MediaPrivacyRepository) GetByMediaID(ctx context.Context, mediaID string) ([]PrivacyEntry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, privacy_level FROM media_privacy WHERE media_id = $1`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PrivacyEntry
	for rows.Next() {
		var e PrivacyEntry
		if err := rows.Scan(&e.UserID, &e.PrivacyLevel); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// AggregatePrivacyLevel computes effective level from per-user choices: any Private -> Private; else Public. Restricted is never downgraded by this (ML sets it).
func AggregatePrivacyLevel(currentLevel string, entries []PrivacyEntry) string {
	if currentLevel == "Restricted" {
		return "Restricted"
	}
	for _, e := range entries {
		if e.PrivacyLevel == "Private" {
			return "Private"
		}
	}
	return "Public"
}
