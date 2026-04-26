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
// SQL guard: never overwrite Restricted ("permanently private", ТЗ 6.3) with a lower level.
func (r *PinRepository) SetPrivacyLevel(pinID, level string) error {
	q := psq.Update("pins").Set("privacy_level", level).Where(sq.Eq{"id": pinID})
	if level != "Restricted" {
		q = q.Where(sq.NotEq{"privacy_level": "Restricted"})
	}
	res, err := q.RunWith(r.db).Exec()
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
// Скрытые пины (pin_hidden_by_user) — отфильтрованы (ТЗ 4.5.2).
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
		LEFT JOIN pin_hidden_by_user ph ON ph.pin_id = p.id AND ph.user_id = $1
		WHERE (p.name ILIKE $2 OR p.description ILIKE $2 OR t.tag ILIKE $2)
		 AND ph.pin_id IS NULL
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
	ID string
	Latitude float64
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

// ListByTripIDExcludingHidden — список пинов трипа, кроме скрытых для userID
// через pin_hidden_by_user (ТЗ 4.5.2 soft-delete-for-self).
func (r *PinRepository) ListByTripIDExcludingHidden(tripID, userID string) ([]*models.Pin, error) {
	sqlStr := `SELECT p.id, p.trip_id, p.name, p.description, p.category, p.privacy_level, p.media_count,
		ST_Y(p.location)::float as lat, ST_X(p.location)::float as lon,
		p.start_time, p.end_time, p.is_published_in_feed, p.location_name, p.created_at
		FROM pins p
		LEFT JOIN pin_hidden_by_user ph ON ph.pin_id = p.id AND ph.user_id = $2
		WHERE p.trip_id = $1 AND ph.pin_id IS NULL
		ORDER BY p.start_time ASC NULLS LAST, p.created_at ASC`
	rows, err := r.db.Query(sqlStr, tripID, userID)
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

// IncMediaCount атомарно увеличивает (delta>0) или уменьшает (delta<0)
// pins.media_count для пина. Используется AddMediaToPin/RemoveMediaFromPin.
// Защищён от ухода в отрицательные значения через GREATEST.
func (r *PinRepository) IncMediaCount(pinID string, delta int) error {
	res, err := r.db.Exec(
		`UPDATE pins SET media_count = GREATEST(media_count + $2, 0) WHERE id = $1`,
		pinID, delta)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RecommendationPinCandidate — пин-кандидат для рекомендательной выборки (ТЗ 9):
// результат CTE с ST_ClusterDBSCAN, партиционированной по category. ClusterID=-1
// проставляется на случай NULL (DBSCAN-edge-cases при minpoints=1 не должен давать
// NULL, но защищаемся sql.NullInt64).
type RecommendationPinCandidate struct {
	ID string
	TripID string
	Name string
	Description string
	Category string
	LocationName string
	MediaCount int32
	Latitude float64
	Longitude float64
	ClusterID int32
	TripScore int32
}

// ListRecommendationCandidates — выборка для ТЗ 9.2:
//   - топ-50 опубликованных трипов региона за 2 года по score = likes - dislikes;
//   - все их пины, опубликованные в фид и с координатами;
//   - кластеризация ST_ClusterDBSCAN по координатам, eps в метрах (через ST_Transform → 3857),
//     отдельные кластеры на каждую категорию (PARTITION BY p.category).
//
// epsMeters: 50 (ТЗ 9.2.3.a) или 500 (9.2.3.b).
func (r *PinRepository) ListRecommendationCandidates(locationID int, epsMeters float64) ([]*RecommendationPinCandidate, error) {
	const sqlStr = `
WITH top_trips AS (
 SELECT t.id, (t.likes_count - t.dislikes_count)::int AS score
 FROM trips t
 JOIN trip_locations tl ON tl.trip_id = t.id
 WHERE tl.location_id = $1
  AND t.is_published = true
  AND t.is_soft_deleted = false
  AND t.privacy_level = 'Public'
  AND COALESCE(t.end_date, t.start_date) >= NOW() - INTERVAL '2 years'
 ORDER BY score DESC
 LIMIT 50
),
candidates AS (
 SELECT p.id, p.trip_id, p.name, COALESCE(p.description, '') AS description,
  p.category, p.location_name, p.media_count,
  ST_Y(p.location)::float AS lat, ST_X(p.location)::float AS lon,
  p.location, tt.score
 FROM pins p
 JOIN top_trips tt ON tt.id = p.trip_id
 WHERE p.is_published_in_feed = true AND p.location IS NOT NULL
)
SELECT id, trip_id, name, description, category, location_name, media_count,
 lat, lon,
 ST_ClusterDBSCAN(ST_Transform(location, 3857), eps := $2, minpoints := 1)
  OVER (PARTITION BY category) AS cluster_id,
 score
FROM candidates`
	rows, err := r.db.Query(sqlStr, locationID, epsMeters)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RecommendationPinCandidate
	for rows.Next() {
		var c RecommendationPinCandidate
		var cluster sql.NullInt64
		if err := rows.Scan(
			&c.ID, &c.TripID, &c.Name, &c.Description, &c.Category, &c.LocationName, &c.MediaCount,
			&c.Latitude, &c.Longitude, &cluster, &c.TripScore,
		); err != nil {
			return nil, err
		}
		if cluster.Valid {
			c.ClusterID = int32(cluster.Int64)
		} else {
			c.ClusterID = -1
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}
