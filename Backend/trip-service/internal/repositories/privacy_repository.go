package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

// PrivacyEntry is one per-user privacy choice.
type PrivacyEntry struct {
	UserID string
	PrivacyLevel string
}

// TripPrivacyRepository handles trip_privacy table (per-user trip privacy).
type TripPrivacyRepository struct{ q *sqlcdb.Queries }

func NewTripPrivacyRepository(db *sql.DB) *TripPrivacyRepository {
	return &TripPrivacyRepository{q: sqlcdb.New(db)}
}

func (r *TripPrivacyRepository) Upsert(ctx context.Context, tripID, userID, privacyLevel string) error {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.TripPrivacyUpsert(ctx, sqlcdb.TripPrivacyUpsertParams{
		TripID: tid,
		UserID: uid,
		PrivacyLevel: privacyLevel,
	})
}

func (r *TripPrivacyRepository) GetByTripID(ctx context.Context, tripID string) ([]PrivacyEntry, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.TripPrivacyListByTrip(ctx, tid)
	if err != nil {
		return nil, err
	}
	out := make([]PrivacyEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, PrivacyEntry{UserID: row.UserID.String(), PrivacyLevel: row.PrivacyLevel})
	}
	return out, nil
}

// PinPrivacyRepository handles pin_privacy table.
type PinPrivacyRepository struct{ q *sqlcdb.Queries }

func NewPinPrivacyRepository(db *sql.DB) *PinPrivacyRepository {
	return &PinPrivacyRepository{q: sqlcdb.New(db)}
}

func (r *PinPrivacyRepository) Upsert(ctx context.Context, pinID, userID, privacyLevel string) error {
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.PinPrivacyUpsert(ctx, sqlcdb.PinPrivacyUpsertParams{
		PinID: pid,
		UserID: uid,
		PrivacyLevel: privacyLevel,
	})
}

func (r *PinPrivacyRepository) GetByPinID(ctx context.Context, pinID string) ([]PrivacyEntry, error) {
	pid, err := uuid.Parse(pinID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.PinPrivacyListByPin(ctx, pid)
	if err != nil {
		return nil, err
	}
	out := make([]PrivacyEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, PrivacyEntry{UserID: row.UserID.String(), PrivacyLevel: row.PrivacyLevel})
	}
	return out, nil
}

// MediaPrivacyRepository handles media_privacy table.
type MediaPrivacyRepository struct{ q *sqlcdb.Queries }

func NewMediaPrivacyRepository(db *sql.DB) *MediaPrivacyRepository {
	return &MediaPrivacyRepository{q: sqlcdb.New(db)}
}

func (r *MediaPrivacyRepository) Upsert(ctx context.Context, mediaID, userID, privacyLevel string) error {
	mid, err := uuid.Parse(mediaID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.q.MediaPrivacyUpsert(ctx, sqlcdb.MediaPrivacyUpsertParams{
		MediaID: mid,
		UserID: uid,
		PrivacyLevel: privacyLevel,
	})
}

func (r *MediaPrivacyRepository) GetByMediaID(ctx context.Context, mediaID string) ([]PrivacyEntry, error) {
	mid, err := uuid.Parse(mediaID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.MediaPrivacyListByMedia(ctx, mid)
	if err != nil {
		return nil, err
	}
	out := make([]PrivacyEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, PrivacyEntry{UserID: row.UserID.String(), PrivacyLevel: row.PrivacyLevel})
	}
	return out, nil
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
