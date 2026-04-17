package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	if m.ContentHash != nil {
		cols = append(cols, "content_hash")
		vals = append(vals, *m.ContentHash)
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
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, created_at
		FROM media WHERE id = $1`
	var m models.Media
	var pinID, similarGroupID, contentHash sql.NullString
	var lat, lon sql.NullFloat64
	var capturedAt sql.NullTime
	err := r.db.QueryRow(sqlStr, id).Scan(&m.ID, &m.TripID, &pinID, &m.S3Key, &m.MediaType,
		&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &contentHash, &m.CreatedAt)
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
	if contentHash.Valid {
		m.ContentHash = &contentHash.String
	}
	return &m, nil
}

func (r *MediaRepository) ListByTripID(tripID string) ([]*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_X(location)::float as lat, ST_Y(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, created_at
		FROM media WHERE trip_id = $1 ORDER BY captured_at ASC NULLS LAST, created_at ASC`
	rows, err := r.db.Query(sqlStr, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Media
	for rows.Next() {
		var m models.Media
		var pinID, similarGroupID, contentHash sql.NullString
		var lat, lon sql.NullFloat64
		var capturedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.TripID, &pinID, &m.S3Key, &m.MediaType,
			&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &contentHash, &m.CreatedAt); err != nil {
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
		if contentHash.Valid {
			m.ContentHash = &contentHash.String
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

// MarkNSFW выставляет privacy_level='Restricted' для переданных медиа (цензура через ML).
func (r *MediaRepository) MarkNSFW(mediaIDs []string) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	_, err := psq.Update("media").Set("privacy_level", "Restricted").Where(sq.Eq{"id": mediaIDs}).RunWith(r.db).Exec()
	return err
}

// SetPrivacyLevel sets privacy_level on a single media (used by privacy aggregation worker).
// SQL guard: never overwrite Restricted ("permanently private", ТЗ 6.3) with a lower level.
func (r *MediaRepository) SetPrivacyLevel(mediaID, level string) error {
	q := psq.Update("media").Set("privacy_level", level).Where(sq.Eq{"id": mediaID})
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

// CountByTripID returns total media count and video count for the trip (task limits: max 500 media, max 50 videos).
func (r *MediaRepository) CountByTripID(tripID string) (total int, videos int, err error) {
	err = r.db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE media_type = 'video') FROM media WHERE trip_id = $1`, tripID).Scan(&total, &videos)
	return total, videos, err
}

// ClusterIDsByLocation returns for each media ID (with location) its cluster index (PostGIS ST_ClusterDBSCAN).
// Uses an azimuthal equidistant (AEQD) projection centred on the trip centroid so that eps is expressed
// directly in metres and distortion stays below 10 m within several-thousand-kilometre trips.
func (r *MediaRepository) ClusterIDsByLocation(tripID string, radiusMeters float64) (map[string]int, error) {
	sqlStr := `
		WITH c AS (
			SELECT ST_Y(ST_Centroid(ST_Collect(location))) AS lat0,
			       ST_X(ST_Centroid(ST_Collect(location))) AS lon0
			FROM media
			WHERE trip_id = $1 AND location IS NOT NULL
		)
		SELECT m.id,
		       ST_ClusterDBSCAN(
		           ST_Transform(
		               m.location,
		               format('+proj=aeqd +lat_0=%s +lon_0=%s +ellps=WGS84 +units=m +no_defs', c.lat0, c.lon0)
		           ),
		           $2,
		           1
		       ) OVER ()::int AS cid
		FROM media m CROSS JOIN c
		WHERE m.trip_id = $1 AND m.location IS NOT NULL`
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
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, created_at
		FROM media WHERE pin_id = $1 ORDER BY captured_at ASC NULLS LAST, created_at ASC`
	rows, err := r.db.Query(sqlStr, pinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Media
	for rows.Next() {
		var m models.Media
		var pinIDNull, similarGroupID, contentHash sql.NullString
		var lat, lon sql.NullFloat64
		var capturedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.TripID, &pinIDNull, &m.S3Key, &m.MediaType,
			&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &contentHash, &m.CreatedAt); err != nil {
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
		if contentHash.Valid {
			m.ContentHash = &contentHash.String
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}

type FeedMedia struct {
	ID        string
	S3Key     string
	MediaType string
}

func (r *MediaRepository) TopMediaByTripIDs(tripIDs []string, limitPerTrip int) (map[string][]*FeedMedia, error) {
	if len(tripIDs) == 0 || limitPerTrip <= 0 {
		return nil, nil
	}
	placeholders := make([]string, len(tripIDs))
	args := make([]interface{}, len(tripIDs)+1)
	for i, id := range tripIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(tripIDs)] = limitPerTrip
	sqlStr := `SELECT id, trip_id, s3_key, media_type FROM (
		SELECT id, trip_id, s3_key, media_type,
			ROW_NUMBER() OVER (PARTITION BY trip_id ORDER BY battle_rating DESC, id) AS rn
		FROM media WHERE trip_id IN (` + strings.Join(placeholders, ",") + `)
	) sub WHERE rn <= $` + fmt.Sprintf("%d", len(tripIDs)+1)
	rows, err := r.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]*FeedMedia)
	for rows.Next() {
		var id, tripID, s3Key, mediaType string
		if err := rows.Scan(&id, &tripID, &s3Key, &mediaType); err != nil {
			return nil, err
		}
		out[tripID] = append(out[tripID], &FeedMedia{ID: id, S3Key: s3Key, MediaType: mediaType})
	}
	return out, rows.Err()
}
