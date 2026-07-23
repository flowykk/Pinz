package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
	"pinz/backend/trip-service/internal/models"
)

type TripParticipantRepository struct {
	q *sqlcdb.Queries
}

func NewTripParticipantRepository(db *sql.DB) *TripParticipantRepository {
	return &TripParticipantRepository{q: sqlcdb.New(db)}
}

func (r *TripParticipantRepository) Add(p *models.TripParticipant) error {
	tid, err := uuid.Parse(p.TripID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(p.UserID)
	if err != nil {
		return err
	}
	return r.q.TripParticipantAdd(context.Background(), sqlcdb.TripParticipantAddParams{
		TripID: tid,
		UserID: uid,
		IsAdmin: p.IsAdmin,
	})
}

func (r *TripParticipantRepository) GetByTripID(tripID string) ([]*models.TripParticipant, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.TripParticipantListByTrip(context.Background(), tid)
	if err != nil {
		return nil, err
	}
	list := make([]*models.TripParticipant, 0, len(rows))
	for _, row := range rows {
		list = append(list, &models.TripParticipant{
			TripID: row.TripID.String(),
			UserID: row.UserID.String(),
			IsAdmin: row.IsAdmin,
			JoinedAt: row.JoinedAt,
		})
	}
	return list, nil
}

func (r *TripParticipantRepository) IsParticipant(tripID, userID string) (bool, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return false, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	_, err = r.q.TripParticipantIsParticipant(context.Background(), sqlcdb.TripParticipantIsParticipantParams{
		TripID: tid,
		UserID: uid,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *TripParticipantRepository) IsAdmin(tripID, userID string) (bool, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return false, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	isAdmin, err := r.q.TripParticipantIsAdmin(context.Background(), sqlcdb.TripParticipantIsAdminParams{
		TripID: tid,
		UserID: uid,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return isAdmin, nil
}

func (r *TripParticipantRepository) Remove(tripID, userID string) error {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	n, err := r.q.TripParticipantRemove(context.Background(), sqlcdb.TripParticipantRemoveParams{TripID: tid, UserID: uid})
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RemoveAllByTripID removes all participants from the trip.
func (r *TripParticipantRepository) RemoveAllByTripID(tripID string) error {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	return r.q.TripParticipantRemoveAllByTrip(context.Background(), tid)
}

// SetAdmin sets the given user as the only admin for the trip (is_admin=true for userID, false for others).
func (r *TripParticipantRepository) SetAdmin(tripID, userID string) error {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	if err := r.q.TripParticipantClearAdmin(context.Background(), tid); err != nil {
		return err
	}
	return r.q.TripParticipantSetAdmin(context.Background(), sqlcdb.TripParticipantSetAdminParams{
		TripID: tid,
		UserID: uid,
	})
}
