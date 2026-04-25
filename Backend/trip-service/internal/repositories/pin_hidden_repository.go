package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

type PinHiddenRepository struct {
	q *sqlcdb.Queries
}

func NewPinHiddenRepository(db *sql.DB) *PinHiddenRepository {
	return &PinHiddenRepository{q: sqlcdb.New(db)}
}

func (r *PinHiddenRepository) HidePinForUser(pinID, userID string) error {
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.PinHiddenInsert(context.Background(), sqlcdb.PinHiddenInsertParams{PinID: pid, UserID: uid})
}

func (r *PinHiddenRepository) ListHiddenPinIDsForUser(tripID, userID string) ([]string, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.PinHiddenListForUser(context.Background(), sqlcdb.PinHiddenListForUserParams{TripID: tid, UserID: uid})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, id := range rows {
		out = append(out, id.String())
	}
	return out, nil
}

// IsHidden проверяет, скрыт ли пин для конкретного юзера (ТЗ 4.5.2).
// Используется в GetPin для возврата 404, как если бы пин не существовал.
func (r *PinHiddenRepository) IsHidden(pinID, userID string) (bool, error) {
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return false, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	hidden, err := r.q.PinHiddenIsHidden(context.Background(), sqlcdb.PinHiddenIsHiddenParams{PinID: pid, UserID: uid})
	if err != nil {
		return false, err
	}
	return hidden, nil
}
