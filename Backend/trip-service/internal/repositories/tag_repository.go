package repositories

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"pinz/backend/trip-service/internal/models"
)

type TagRepository struct {
	db *sql.DB
}

const (
	maxTagsPerPin = 10
	maxTagLength  = 15
)

func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) SetForPin(tripID, pinID string, tags []string) error {
	if len(tags) > maxTagsPerPin {
		tags = tags[:maxTagsPerPin]
	}
	if _, err := psq.Delete("tags").Where(sq.Eq{"trip_id": tripID, "pin_id": pinID}).RunWith(r.db).Exec(); err != nil {
		return err
	}
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if len(tag) > maxTagLength {
			tag = tag[:maxTagLength]
		}
		t := &models.Tag{TripID: tripID, PinID: pinID, Tag: tag}
		if err := r.Add(t); err != nil {
			return err
		}
	}
	return nil
}

func (r *TagRepository) Add(t *models.Tag) error {
	if t.Tag == "" {
		return nil
	}
	if len(t.Tag) > maxTagLength {
		t.Tag = t.Tag[:maxTagLength]
	}
	q := psq.Insert("tags").Columns("trip_id", "pin_id", "tag").Values(t.TripID, t.PinID, t.Tag).Suffix("RETURNING id")
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return err
	}
	return r.db.QueryRow(sqlStr, args...).Scan(&t.ID)
}

func (r *TagRepository) GetByPinID(pinID string) ([]string, error) {
	rows, err := psq.Select("tag").From("tags").Where(sq.Eq{"pin_id": pinID}).RunWith(r.db).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *TagRepository) GetByTripID(tripID string) (map[string][]string, error) {
	rows, err := psq.Select("pin_id", "tag").From("tags").Where(sq.Eq{"trip_id": tripID}).RunWith(r.db).Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]string)
	for rows.Next() {
		var pinID, tag string
		if err := rows.Scan(&pinID, &tag); err != nil {
			return nil, err
		}
		out[pinID] = append(out[pinID], tag)
	}
	return out, rows.Err()
}

func (r *TagRepository) DeleteForPin(pinID string) error {
	_, err := psq.Delete("tags").Where(sq.Eq{"pin_id": pinID}).RunWith(r.db).Exec()
	return err
}
