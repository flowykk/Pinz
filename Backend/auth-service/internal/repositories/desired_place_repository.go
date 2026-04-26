package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"pinz/backend/auth-service/internal/db/sqlcdb"
	"pinz/backend/auth-service/internal/models"
)

type DesiredPlaceRepository struct {
	q *sqlcdb.Queries
}

func NewDesiredPlaceRepository(db *sql.DB) *DesiredPlaceRepository {
	return &DesiredPlaceRepository{q: sqlcdb.New(db)}
}

func (r *DesiredPlaceRepository) Create(p *models.DesiredPlace) (*models.DesiredPlace, error) {
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(p.UserID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateDesiredPlace(context.Background(), sqlcdb.CreateDesiredPlaceParams{
		ID:          id,
		UserID:      uid,
		Name:        p.Name,
		Description: p.Description,
		ImageUrl:    sql.NullString{String: p.ImageURL, Valid: p.ImageURL != ""},
	})
	if err != nil {
		return nil, err
	}
	return desiredPlaceFromSQLC(row), nil
}

func (r *DesiredPlaceRepository) GetByID(placeID string) (*models.DesiredPlace, error) {
	id, err := uuid.Parse(placeID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetDesiredPlace(context.Background(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return desiredPlaceFromSQLC(row), nil
}

func (r *DesiredPlaceRepository) ListByUserID(userID string) ([]*models.DesiredPlace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListDesiredPlacesByUserID(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	out := make([]*models.DesiredPlace, 0, len(rows))
	for _, row := range rows {
		out = append(out, desiredPlaceFromSQLC(row))
	}
	return out, nil
}

// Update перезаписывает name/description и заменяет image_url на новое значение
// (пустая строка = NULL в БД).
func (r *DesiredPlaceRepository) Update(placeID, name, description, imageURL string) (*models.DesiredPlace, error) {
	id, err := uuid.Parse(placeID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateDesiredPlace(context.Background(), sqlcdb.UpdateDesiredPlaceParams{
		ID:          id,
		Name:        name,
		Description: description,
		ImageUrl:    sql.NullString{String: imageURL, Valid: imageURL != ""},
	})
	if err != nil {
		return nil, err
	}
	return desiredPlaceFromSQLC(row), nil
}

// UpdateContent меняет только name и description, image_url не трогает.
func (r *DesiredPlaceRepository) UpdateContent(placeID, name, description string) (*models.DesiredPlace, error) {
	id, err := uuid.Parse(placeID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateDesiredPlaceContent(context.Background(), sqlcdb.UpdateDesiredPlaceContentParams{
		ID:          id,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, err
	}
	return desiredPlaceFromSQLC(row), nil
}

func (r *DesiredPlaceRepository) ClearImage(placeID string) (*models.DesiredPlace, error) {
	id, err := uuid.Parse(placeID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.ClearDesiredPlaceImage(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return desiredPlaceFromSQLC(row), nil
}

func (r *DesiredPlaceRepository) Delete(placeID string) error {
	id, err := uuid.Parse(placeID)
	if err != nil {
		return err
	}
	return r.q.DeleteDesiredPlace(context.Background(), id)
}

func desiredPlaceFromSQLC(p sqlcdb.DesiredPlace) *models.DesiredPlace {
	out := &models.DesiredPlace{
		ID:          p.ID.String(),
		UserID:      p.UserID.String(),
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
	}
	if p.ImageUrl.Valid {
		out.ImageURL = p.ImageUrl.String
	}
	return out
}
