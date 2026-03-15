package repositories

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"

	"pinz/backend/trip-service/internal/models"
)

type PinRepository struct {
	db *sql.DB
}

func NewPinRepository(db *sql.DB) *PinRepository {
	return &PinRepository{db: db}
}

func (r *PinRepository) Create(p *models.Pin) error {
	cols := []string{"trip_id", "name", "description", "category", "privacy_level", "media_count"}
	vals := []interface{}{p.TripID, p.Name, p.Description, p.Category, p.PrivacyLevel, p.MediaCount}
	if p.Latitude != nil && p.Longitude != nil {
		cols = append(cols, "location")
		vals = append(vals, sq.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)", *p.Longitude, *p.Latitude))
	}
	if p.StartTime != nil {
		cols = append(cols, "start_time")
		vals = append(vals, p.StartTime)
	}
	if p.EndTime != nil {
		cols = append(cols, "end_time")
		vals = append(vals, p.EndTime)
	}
	q := psq.Insert("pins").Columns(cols...).Values(vals...).Suffix("RETURNING id, created_at")
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return err
	}
	err = r.db.QueryRow(sqlStr, args...).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *PinRepository) GetByID(id string) (*models.Pin, error) {
	sqlStr := `SELECT id, trip_id, name, description, category, privacy_level, media_count,
		ST_X(location)::float as lat, ST_Y(location)::float as lon,
		start_time, end_time, created_at
		FROM pins WHERE id = $1`
	var p models.Pin
	var desc sql.NullString
	var lat, lon sql.NullFloat64
	var startTime, endTime sql.NullTime
	err := r.db.QueryRow(sqlStr, id).Scan(&p.ID, &p.TripID, &p.Name, &desc, &p.Category, &p.PrivacyLevel, &p.MediaCount,
		&lat, &lon, &startTime, &endTime, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if desc.Valid {
		p.Description = desc.String
	}
	if lat.Valid {
		p.Latitude = &lat.Float64
	}
	if lon.Valid {
		p.Longitude = &lon.Float64
	}
	if startTime.Valid {
		p.StartTime = &startTime.Time
	}
	if endTime.Valid {
		p.EndTime = &endTime.Time
	}
	return &p, nil
}

func (r *PinRepository) ListByTripID(tripID string) ([]*models.Pin, error) {
	sqlStr := `SELECT id, trip_id, name, description, category, privacy_level, media_count,
		ST_X(location)::float as lat, ST_Y(location)::float as lon,
		start_time, end_time, created_at
		FROM pins WHERE trip_id = $1 ORDER BY start_time ASC NULLS LAST, created_at ASC`
	rows, err := r.db.Query(sqlStr, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Pin
	for rows.Next() {
		var p models.Pin
		var desc sql.NullString
		var lat, lon sql.NullFloat64
		var startTime, endTime sql.NullTime
		if err := rows.Scan(&p.ID, &p.TripID, &p.Name, &desc, &p.Category, &p.PrivacyLevel, &p.MediaCount,
			&lat, &lon, &startTime, &endTime, &p.CreatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			p.Description = desc.String
		}
		if lat.Valid {
			p.Latitude = &lat.Float64
		}
		if lon.Valid {
			p.Longitude = &lon.Float64
		}
		if startTime.Valid {
			p.StartTime = &startTime.Time
		}
		if endTime.Valid {
			p.EndTime = &endTime.Time
		}
		list = append(list, &p)
	}
	return list, rows.Err()
}

func (r *PinRepository) Update(p *models.Pin) error {
	u := psq.Update("pins").
		Set("name", p.Name).
		Set("description", p.Description).
		Set("category", p.Category).
		Set("privacy_level", p.PrivacyLevel).
		Set("media_count", p.MediaCount)
	if p.Latitude != nil && p.Longitude != nil {
		u = u.Set("location", sq.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)", *p.Longitude, *p.Latitude))
	} else {
		u = u.Set("location", nil)
	}
	if p.StartTime != nil {
		u = u.Set("start_time", p.StartTime)
	} else {
		u = u.Set("start_time", nil)
	}
	if p.EndTime != nil {
		u = u.Set("end_time", p.EndTime)
	} else {
		u = u.Set("end_time", nil)
	}
	u = u.Where(sq.Eq{"id": p.ID})
	sqlStr, args, err := u.ToSql()
	if err != nil {
		return err
	}
	res, err := r.db.Exec(sqlStr, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PinRepository) Delete(id string) error {
	res, err := psq.Delete("pins").Where(sq.Eq{"id": id}).RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PinRepository) DeleteByTripID(tripID string) error {
	_, err := psq.Delete("pins").Where(sq.Eq{"trip_id": tripID}).RunWith(r.db).Exec()
	return err
}
