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

// GetByTripAndUsers returns the notifications_enabled flag for each user in
// the trip. Users without a row in trip_settings default to true (matches
// EnsureDefaultSettings). Used by notification-service via the
// GetNotificationSettings RPC.
func (r *TripSettingsRepository) GetByTripAndUsers(tripID string, userIDs []string) (map[string]bool, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, err
	}
	uids := make([]uuid.UUID, 0, len(userIDs))
	for _, s := range userIDs {
		u, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		uids = append(uids, u)
	}
	out := make(map[string]bool, len(userIDs))
	for _, s := range userIDs {
		out[s] = true
	}
	if len(uids) == 0 {
		return out, nil
	}
	rows, err := r.q.TripSettingsGetByTripAndUsers(context.Background(), sqlcdb.TripSettingsGetByTripAndUsersParams{
		TripID:  tid,
		Column2: uids,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.UserID.String()] = row.NotificationsEnabled
	}
	return out, nil
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
