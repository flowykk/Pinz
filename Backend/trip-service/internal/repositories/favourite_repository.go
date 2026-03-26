package repositories

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
)

type FavouriteRepository struct {
	db *sql.DB
}

func NewFavouriteRepository(db *sql.DB) *FavouriteRepository {
	return &FavouriteRepository{db: db}
}

// Add adds a trip to user's favourites. Idempotent.
func (r *FavouriteRepository) Add(userID, tripID string) error {
	_, err := psq.Insert("favourite").
		Columns("user_id", "trip_id").
		Values(userID, tripID).
		Suffix("ON CONFLICT (user_id, trip_id) DO NOTHING").
		RunWith(r.db).Exec()
	return err
}

// Remove removes a trip from user's favourites.
func (r *FavouriteRepository) Remove(userID, tripID string) error {
	res, err := psq.Delete("favourite").
		Where(sq.Eq{"user_id": userID, "trip_id": tripID}).
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

// HasFavourite returns true if user has the trip in favourites.
func (r *FavouriteRepository) HasFavourite(userID, tripID string) (bool, error) {
	var d int
	err := r.db.QueryRow("SELECT 1 FROM favourite WHERE user_id = $1 AND trip_id = $2 LIMIT 1", userID, tripID).Scan(&d)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// HasFavouritesByOtherUsers returns true if any user other than excludeUserID has this trip in favourites.
func (r *FavouriteRepository) HasFavouritesByOtherUsers(tripID, excludeUserID string) (bool, error) {
	var d int
	err := r.db.QueryRow(
		"SELECT 1 FROM favourite WHERE trip_id = $1 AND user_id != $2 LIMIT 1",
		tripID, excludeUserID,
	).Scan(&d)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListTripIDsByUserID returns trip IDs for the user's favourites, ordered by created_at DESC (newest first).
func (r *FavouriteRepository) ListTripIDsByUserID(userID string, limit, offset int32) ([]string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := psq.Select("trip_id").
		From("favourite").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		RunWith(r.db).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
