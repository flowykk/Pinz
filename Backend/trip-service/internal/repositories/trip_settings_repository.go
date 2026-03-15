package repositories

import "database/sql"

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
