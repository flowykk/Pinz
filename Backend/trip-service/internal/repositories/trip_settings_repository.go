package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

type TripSettingsRepository struct {
	q *sqlcdb.Queries
}

func NewTripSettingsRepository(db *sql.DB) *TripSettingsRepository {
	return &TripSettingsRepository{q: sqlcdb.New(db)}
}

// EnsureDefaultSettings creates a trip_settings row with notifications_enabled=true.
// Idempotent: safe to call when joining a trip (duplicate key is ignored).
func (r *TripSettingsRepository) EnsureDefaultSettings(tripID, userID string) error {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.TripSettingsEnsureDefault(context.Background(), sqlcdb.TripSettingsEnsureDefaultParams{
		UserID: uid,
		TripID: tid,
	})
}

// UpdateNotifications updates notifications_enabled for the user's trip settings.
func (r *TripSettingsRepository) UpdateNotifications(tripID, userID string, enabled bool) error {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	n, err := r.q.TripSettingsUpdateNotifications(context.Background(), sqlcdb.TripSettingsUpdateNotificationsParams{
		NotificationsEnabled: enabled,
		TripID:               tid,
		UserID:               uid,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
