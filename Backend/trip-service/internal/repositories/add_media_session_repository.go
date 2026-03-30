package repositories

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

type AddMediaSessionRepository struct {
	q *sqlcdb.Queries
}

func NewAddMediaSessionRepository(db *sql.DB) *AddMediaSessionRepository {
	return &AddMediaSessionRepository{q: sqlcdb.New(db)}
}

func (r *AddMediaSessionRepository) Create(ctx context.Context, tripID string, existingMediaIDs []string) (sessionID string, err error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(existingMediaIDs)
	if err != nil {
		return "", err
	}
	sid, err := r.q.AddMediaSessionCreate(ctx, sqlcdb.AddMediaSessionCreateParams{
		TripID:           tid,
		ExistingMediaIds: b,
	})
	if err != nil {
		return "", err
	}
	return sid.String(), nil
}

func (r *AddMediaSessionRepository) GetExistingMediaIDs(ctx context.Context, sessionID string) ([]string, string, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, "", err
	}
	row, err := r.q.AddMediaSessionGet(ctx, sid)
	if err != nil {
		return nil, "", err
	}
	var ids []string
	if err := json.Unmarshal(row.ExistingMediaIds, &ids); err != nil {
		return nil, "", err
	}
	return ids, row.TripID.String(), nil
}

func (r *AddMediaSessionRepository) Exists(ctx context.Context, tripID, sessionID string) (bool, error) {
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return false, err
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return false, err
	}
	n, err := r.q.AddMediaSessionExists(ctx, sqlcdb.AddMediaSessionExistsParams{TripID: tid, SessionID: sid})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
