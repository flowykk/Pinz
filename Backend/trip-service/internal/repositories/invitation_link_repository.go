package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
	"pinz/backend/trip-service/internal/models"
)

type InvitationLinkRepository struct {
	q *sqlcdb.Queries
}

func NewInvitationLinkRepository(db *sql.DB) *InvitationLinkRepository {
	return &InvitationLinkRepository{q: sqlcdb.New(db)}
}

func (r *InvitationLinkRepository) Create(link *models.InvitationLink) error {
	id, err := uuid.Parse(link.ID)
	if err != nil {
		return err
	}
	tid, err := uuid.Parse(link.TripID)
	if err != nil {
		return err
	}
	return r.q.InvitationLinkInsert(context.Background(), sqlcdb.InvitationLinkInsertParams{
		ID: id,
		TripID: tid,
		Token: link.Token,
		ExpiresAt: link.ExpiresAt,
	})
}

func (r *InvitationLinkRepository) GetByToken(token string) (*models.InvitationLink, error) {
	row, err := r.q.InvitationLinkByToken(context.Background(), token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &models.InvitationLink{
		ID: row.ID.String(),
		TripID: row.TripID.String(),
		Token: row.Token,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}
