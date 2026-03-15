package repositories

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"

	"pinz/backend/trip-service/internal/models"
)

type TripParticipantRepository struct {
	db *sql.DB
}

func NewTripParticipantRepository(db *sql.DB) *TripParticipantRepository {
	return &TripParticipantRepository{db: db}
}

func (r *TripParticipantRepository) Add(p *models.TripParticipant) error {
	_, err := psq.Insert("trip_participants").
		Columns("trip_id", "user_id", "is_admin").
		Values(p.TripID, p.UserID, p.IsAdmin).
		RunWith(r.db).Exec()
	return err
}

func (r *TripParticipantRepository) GetByTripID(tripID string) ([]*models.TripParticipant, error) {
	sqlStr, args, err := psq.Select("trip_id", "user_id", "is_admin", "joined_at").
		From("trip_participants").
		Where(sq.Eq{"trip_id": tripID}).
		ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.TripParticipant
	for rows.Next() {
		var p models.TripParticipant
		if err := rows.Scan(&p.TripID, &p.UserID, &p.IsAdmin, &p.JoinedAt); err != nil {
			return nil, err
		}
		list = append(list, &p)
	}
	return list, rows.Err()
}

func (r *TripParticipantRepository) IsParticipant(tripID, userID string) (bool, error) {
	sqlStr, args, err := psq.Select("1").
		From("trip_participants").
		Where(sq.Eq{"trip_id": tripID, "user_id": userID}).
		Limit(1).ToSql()
	if err != nil {
		return false, err
	}
	var d int
	err = r.db.QueryRow(sqlStr, args...).Scan(&d)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *TripParticipantRepository) IsAdmin(tripID, userID string) (bool, error) {
	sqlStr, args, err := psq.Select("is_admin").
		From("trip_participants").
		Where(sq.Eq{"trip_id": tripID, "user_id": userID}).
		ToSql()
	if err != nil {
		return false, err
	}
	var isAdmin bool
	err = r.db.QueryRow(sqlStr, args...).Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return isAdmin, nil
}

func (r *TripParticipantRepository) Remove(tripID, userID string) error {
	res, err := psq.Delete("trip_participants").
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

// SetAdmin sets the given user as the only admin for the trip (is_admin=true for userID, false for others).
func (r *TripParticipantRepository) SetAdmin(tripID, userID string) error {
	_, err := psq.Update("trip_participants").
		Set("is_admin", false).
		Where(sq.Eq{"trip_id": tripID}).
		RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	_, err = psq.Update("trip_participants").
		Set("is_admin", true).
		Where(sq.Eq{"trip_id": tripID, "user_id": userID}).
		RunWith(r.db).Exec()
	return err
}
