package repositories

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
)

type TripSettingsRepository struct {
	db *sql.DB
}

func NewTripSettingsRepository(db *sql.DB) *TripSettingsRepository {
	return &TripSettingsRepository{db: db}
}

// EnsureDefaultSettings creates a trip_settings row with notifications_enabled=true.
// Idempotent: safe to call when joining a trip (duplicate key is ignored).
func (r *TripSettingsRepository) EnsureDefaultSettings(tripID, userID string) error {
	_, err := psq.Insert("trip_settings").
		Columns("user_id", "trip_id", "notifications_enabled").
		Values(userID, tripID, true).
		Suffix("ON CONFLICT (user_id, trip_id) DO NOTHING").
		RunWith(r.db).Exec()
	return err
}

// UpdateNotifications updates notifications_enabled for the user's trip settings.
func (r *TripSettingsRepository) UpdateNotifications(tripID, userID string, enabled bool) error {
	res, err := psq.Update("trip_settings").
		Set("notifications_enabled", enabled).
		Where(sq.Eq{"trip_id": tripID, "user_id": userID}).
		RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
