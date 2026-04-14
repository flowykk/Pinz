package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	cols := []string{"trip_id", "name", "description", "category", "privacy_level", "media_count", "is_published_in_feed", "location_name"}
	vals := []interface{}{p.TripID, p.Name, p.Description, p.Category, p.PrivacyLevel, p.MediaCount, p.IsPublishedInFeed, p.LocationName}
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
		ST_Y(location)::float as lat, ST_X(location)::float as lon,
		start_time, end_time, is_published_in_feed, location_name, created_at
		FROM pins WHERE id = $1`
	var p models.Pin
	var desc sql.NullString
	var lat, lon sql.NullFloat64
	var startTime, endTime sql.NullTime
	var isPublished sql.NullBool
	err := r.db.QueryRow(sqlStr, id).Scan(&p.ID, &p.TripID, &p.Name, &desc, &p.Category, &p.PrivacyLevel, &p.MediaCount,
		&lat, &lon, &startTime, &endTime, &isPublished, &p.LocationName, &p.CreatedAt)
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
	if isPublished.Valid {
		p.IsPublishedInFeed = isPublished.Bool
	}
	return &p, nil
}

func (r *PinRepository) ListByTripID(tripID string) ([]*models.Pin, error) {
	sqlStr := `SELECT id, trip_id, name, description, category, privacy_level, media_count,
		ST_Y(location)::float as lat, ST_X(location)::float as lon,
		start_time, end_time, is_published_in_feed, location_name, created_at
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
		var isPublished sql.NullBool
		if err := rows.Scan(&p.ID, &p.TripID, &p.Name, &desc, &p.Category, &p.PrivacyLevel, &p.MediaCount,
			&lat, &lon, &startTime, &endTime, &isPublished, &p.LocationName, &p.CreatedAt); err != nil {
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
		if isPublished.Valid {
			p.IsPublishedInFeed = isPublished.Bool
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
		Set("media_count", p.MediaCount).
		Set("is_published_in_feed", p.IsPublishedInFeed).
		Set("location_name", p.LocationName)
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

// SetPrivacyLevel updates only pin privacy_level (used by privacy aggregation worker).
func (r *PinRepository) SetPrivacyLevel(pinID, level string) error {
	res, err := psq.Update("pins").Set("privacy_level", level).Where(sq.Eq{"id": pinID}).RunWith(r.db).Exec()
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

// SearchByUserID returns pins from trips where user is participant, matching query in name, description or tag.
func (r *PinRepository) SearchByUserID(userID, query string, limit, offset int32) ([]*models.Pin, error) {
	if query == "" {
		return nil, nil
	}
	pattern := "%" + query + "%"
	sqlStr := `SELECT DISTINCT ON (p.id) p.id, p.trip_id, p.name, p.description, p.category, p.privacy_level, p.media_count,
		ST_Y(p.location)::float as lat, ST_X(p.location)::float as lon,
		p.start_time, p.end_time, p.is_published_in_feed, p.location_name, p.created_at
		FROM pins p
		INNER JOIN trip_participants tp ON tp.trip_id = p.trip_id AND tp.user_id = $1
		LEFT JOIN tags t ON t.pin_id = p.id AND t.trip_id = p.trip_id
		WHERE p.name ILIKE $2 OR p.description ILIKE $2 OR t.tag ILIKE $2
		ORDER BY p.id, p.created_at DESC
		LIMIT $3 OFFSET $4`
	rows, err := r.db.Query(sqlStr, userID, pattern, limit, offset)
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
		var isPublished sql.NullBool
		if err := rows.Scan(&p.ID, &p.TripID, &p.Name, &desc, &p.Category, &p.PrivacyLevel, &p.MediaCount,
			&lat, &lon, &startTime, &endTime, &isPublished, &p.LocationName, &p.CreatedAt); err != nil {
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
		if isPublished.Valid {
			p.IsPublishedInFeed = isPublished.Bool
		}
		list = append(list, &p)
	}
	return list, rows.Err()
}

type FeedPin struct {
	ID        string
	Latitude  float64
	Longitude float64
}

func (r *PinRepository) ListPublishedPinsByTripIDs(tripIDs []string) (map[string][]*FeedPin, error) {
	if len(tripIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(tripIDs))
	args := make([]interface{}, len(tripIDs))
	for i, id := range tripIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	sqlStr := `SELECT id, trip_id, ST_Y(location)::float as lat, ST_X(location)::float as lon
		FROM pins WHERE trip_id IN (` + strings.Join(placeholders, ",") + `) AND is_published_in_feed = true AND location IS NOT NULL`
	rows, err := r.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]*FeedPin)
	for rows.Next() {
		var id, tripID string
		var lat, lon float64
		if err := rows.Scan(&id, &tripID, &lat, &lon); err != nil {
			return nil, err
		}
		out[tripID] = append(out[tripID], &FeedPin{ID: id, Latitude: lat, Longitude: lon})
	}
	return out, rows.Err()
}
