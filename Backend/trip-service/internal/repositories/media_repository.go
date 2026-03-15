package repositories

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"

	"pinz/backend/trip-service/internal/models"
)

type MediaRepository struct {
	db *sql.DB
}

func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) Create(m *models.Media) error {
	cols := []string{"trip_id", "s3_key", "media_type", "captured_at", "battle_rating", "privacy_level"}
	vals := []interface{}{m.TripID, m.S3Key, m.MediaType, m.CapturedAt, m.BattleRating, m.PrivacyLevel}
	if m.PinID != nil {
		cols = append(cols, "pin_id")
		vals = append(vals, *m.PinID)
	}
	if m.SimilarGroupID != nil {
		cols = append(cols, "similar_group_id")
		vals = append(vals, *m.SimilarGroupID)
	}
	if m.Latitude != nil && m.Longitude != nil {
		cols = append(cols, "location")
		vals = append(vals, sq.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)", *m.Longitude, *m.Latitude))
	}
	q := psq.Insert("media").Columns(cols...).Values(vals...).Suffix("RETURNING id, created_at")
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return err
	}
	err = r.db.QueryRow(sqlStr, args...).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *MediaRepository) GetByID(id string) (*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_X(location)::float as lat, ST_Y(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, created_at
		FROM media WHERE id = $1`
	var m models.Media
	var pinID, similarGroupID sql.NullString
	var lat, lon sql.NullFloat64
	var capturedAt sql.NullTime
	err := r.db.QueryRow(sqlStr, id).Scan(&m.ID, &m.TripID, &pinID, &m.S3Key, &m.MediaType,
		&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if pinID.Valid {
		m.PinID = &pinID.String
	}
	if lat.Valid {
		m.Latitude = &lat.Float64
	}
	if lon.Valid {
		m.Longitude = &lon.Float64
	}
	if capturedAt.Valid {
		m.CapturedAt = &capturedAt.Time
	}
	if similarGroupID.Valid {
		m.SimilarGroupID = &similarGroupID.String
	}
	return &m, nil
}

func (r *MediaRepository) ListByTripID(tripID string) ([]*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_X(location)::float as lat, ST_Y(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, created_at
		FROM media WHERE trip_id = $1 ORDER BY captured_at ASC NULLS LAST, created_at ASC`
	rows, err := r.db.Query(sqlStr, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Media
	for rows.Next() {
		var m models.Media
		var pinID, similarGroupID sql.NullString
		var lat, lon sql.NullFloat64
		var capturedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.TripID, &pinID, &m.S3Key, &m.MediaType,
			&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &m.CreatedAt); err != nil {
			return nil, err
		}
		if pinID.Valid {
			m.PinID = &pinID.String
		}
		if lat.Valid {
			m.Latitude = &lat.Float64
		}
		if lon.Valid {
			m.Longitude = &lon.Float64
		}
		if capturedAt.Valid {
			m.CapturedAt = &capturedAt.Time
		}
		if similarGroupID.Valid {
			m.SimilarGroupID = &similarGroupID.String
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}

func (r *MediaRepository) UpdatePinID(mediaID, pinID string) error {
	res, err := psq.Update("media").Set("pin_id", pinID).Where(sq.Eq{"id": mediaID}).RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *MediaRepository) UpdatePinIDByIDs(mediaIDs []string, pinID string) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	_, err := psq.Update("media").Set("pin_id", pinID).Where(sq.Eq{"id": mediaIDs}).RunWith(r.db).Exec()
	return err
}

func (r *MediaRepository) DeleteByIDs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := psq.Delete("media").Where(sq.Eq{"id": ids}).RunWith(r.db).Exec()
	return err
}

// SetSimilarGroupID помечает медиа как одну группу похожих внутри пина (один и тот же groupID = одна группа).
func (r *MediaRepository) SetSimilarGroupID(mediaIDs []string, groupID string) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	_, err := psq.Update("media").Set("similar_group_id", groupID).Where(sq.Eq{"id": mediaIDs}).RunWith(r.db).Exec()
	return err
}

// CountByTripID returns total media count and video count for the trip (task limits: max 500 media, max 50 videos).
func (r *MediaRepository) CountByTripID(tripID string) (total int, videos int, err error) {
	err = r.db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE media_type = 'video') FROM media WHERE trip_id = $1`, tripID).Scan(&total, &videos)
	return total, videos, err
}

// ClusterIDsByLocation returns for each media ID (with location) its cluster index (PostGIS ST_ClusterDBSCAN).
func (r *MediaRepository) ClusterIDsByLocation(tripID string, radiusMeters float64) (map[string]int, error) {
	sqlStr := `SELECT id, ST_ClusterDBSCAN(location::geography, $2, 1) OVER ()::int as cid FROM media WHERE trip_id = $1 AND location IS NOT NULL`
	rows, err := r.db.Query(sqlStr, tripID, radiusMeters)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var id string
		var cid int
		if err := rows.Scan(&id, &cid); err != nil {
			return nil, err
		}
		out[id] = cid
	}
	return out, rows.Err()
}

// ListByPinID returns media for a pin (for computing pin start/end time and location).
func (r *MediaRepository) ListByPinID(pinID string) ([]*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_X(location)::float as lat, ST_Y(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, created_at
		FROM media WHERE pin_id = $1 ORDER BY captured_at ASC NULLS LAST, created_at ASC`
	rows, err := r.db.Query(sqlStr, pinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Media
	for rows.Next() {
		var m models.Media
		var pinIDNull, similarGroupID sql.NullString
		var lat, lon sql.NullFloat64
		var capturedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.TripID, &pinIDNull, &m.S3Key, &m.MediaType,
			&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &m.CreatedAt); err != nil {
			return nil, err
		}
		if pinIDNull.Valid {
			m.PinID = &pinIDNull.String
		}
		if lat.Valid {
			m.Latitude = &lat.Float64
		}
		if lon.Valid {
			m.Longitude = &lon.Float64
		}
		if capturedAt.Valid {
			m.CapturedAt = &capturedAt.Time
		}
		if similarGroupID.Valid {
			m.SimilarGroupID = &similarGroupID.String
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}
