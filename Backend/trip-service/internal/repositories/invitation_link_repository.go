package repositories

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"

	"pinz/backend/trip-service/internal/models"
)

type InvitationLinkRepository struct {
	db *sql.DB
}

func NewInvitationLinkRepository(db *sql.DB) *InvitationLinkRepository {
	return &InvitationLinkRepository{db: db}
}

func (r *InvitationLinkRepository) Create(link *models.InvitationLink) error {
	_, err := psq.Insert("invitation_links").
		Columns("id", "trip_id", "token", "expires_at").
		Values(link.ID, link.TripID, link.Token, link.ExpiresAt).
		RunWith(r.db).Exec()
	return err
}

func (r *InvitationLinkRepository) GetByToken(token string) (*models.InvitationLink, error) {
	q := psq.Select("id", "trip_id", "token", "expires_at", "created_at").
		From("invitation_links").
		Where(sq.Eq{"token": token})
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRow(sqlStr, args...)
	var link models.InvitationLink
	err = row.Scan(&link.ID, &link.TripID, &link.Token, &link.ExpiresAt, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &link, nil
}
