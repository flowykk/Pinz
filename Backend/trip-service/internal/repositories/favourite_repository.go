package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

type FavouriteRepository struct {
	q *sqlcdb.Queries
}

func NewFavouriteRepository(db *sql.DB) *FavouriteRepository {
	return &FavouriteRepository{q: sqlcdb.New(db)}
}

// Add adds a trip to user's favourites. Idempotent.
func (r *FavouriteRepository) Add(userID, tripID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	return r.q.FavouriteAdd(context.Background(), sqlcdb.FavouriteAddParams{UserID: uid, TripID: tid})
}

// Remove removes a trip from user's favourites.
func (r *FavouriteRepository) Remove(userID, tripID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	n, err := r.q.FavouriteRemove(context.Background(), sqlcdb.FavouriteRemoveParams{UserID: uid, TripID: tid})
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// HasFavourite returns true if user has the trip in favourites.
func (r *FavouriteRepository) HasFavourite(userID, tripID string) (bool, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return false, err
	}
	_, err = r.q.FavouriteHas(context.Background(), sqlcdb.FavouriteHasParams{UserID: uid, TripID: tid})
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
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return false, err
	}
	ex, err := uuid.Parse(excludeUserID)
	if err != nil {
		return false, err
	}
	_, err = r.q.FavouriteHasByOtherUsers(context.Background(), sqlcdb.FavouriteHasByOtherUsersParams{TripID: tid, UserID: ex})
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
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	uuids, err := r.q.FavouriteListTripIDsByUser(context.Background(), sqlcdb.FavouriteListTripIDsByUserParams{
		UserID: uid,
		Limit: limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(uuids))
	for _, id := range uuids {
		out = append(out, id.String())
	}
	return out, nil
}
