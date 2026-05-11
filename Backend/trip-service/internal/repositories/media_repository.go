package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"

	"pinz/backend/trip-service/internal/models"
)

// ErrMediaLimitExceeded возвращается CommitInSession, если INSERT упёрся в общий
// лимит медиа на трип (MaxMediaPerTrip).
var ErrMediaLimitExceeded = errors.New("media limit exceeded")

// ErrVideoLimitExceeded возвращается CommitInSession, если INSERT упёрся в лимит
// видео на трип (MaxVideosPerTrip).
var ErrVideoLimitExceeded = errors.New("video limit exceeded")

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
	if m.UploadedBy != nil {
		cols = append(cols, "uploaded_by")
		vals = append(vals, *m.UploadedBy)
	}
	if m.UploadSessionID != nil {
		cols = append(cols, "upload_session_id")
		vals = append(vals, *m.UploadSessionID)
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

// CommitInSession атомарно вставляет media в рамках add-media сессии: берёт
// advisory lock по session_id (сериализует параллельные commit'ы в ту же сессию),
// проверяет лимиты, делает INSERT и возвращает totalAfter/videosAfter — сколько
// медиа теперь в трипе (нужно сервису, чтобы посчитать media_count_in_session и
// remaining_slots без гонки).
// При превышении лимита возвращает ErrMediaLimitExceeded или ErrVideoLimitExceeded
// с заполненным totalAfter (= total до попытки, INSERT не выполняется).
func (r *MediaRepository) CommitInSession(ctx context.Context, m *models.Media, sessionID string, maxMedia, maxVideos int) (totalAfter, videosAfter int, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", sessionID); err != nil {
		return 0, 0, err
	}
	var total, videos int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(*) FILTER (WHERE media_type = 'video') FROM media WHERE trip_id = $1", m.TripID).Scan(&total, &videos); err != nil {
		return 0, 0, err
	}
	if total >= maxMedia {
		return total, videos, ErrMediaLimitExceeded
	}
	if m.MediaType == "video" && videos >= maxVideos {
		return total, videos, ErrVideoLimitExceeded
	}
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
	if m.UploadedBy != nil {
		cols = append(cols, "uploaded_by")
		vals = append(vals, *m.UploadedBy)
	}
	q := psq.Insert("media").Columns(cols...).Values(vals...).Suffix("RETURNING id, created_at")
	sqlStr, args, qerr := q.ToSql()
	if qerr != nil {
		return 0, 0, qerr
	}
	if err = tx.QueryRowContext(ctx, sqlStr, args...).Scan(&m.ID, &m.CreatedAt); err != nil {
		return 0, 0, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE add_media_sessions SET last_activity_at = NOW() WHERE session_id = $1", sessionID); err != nil {
		return 0, 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	committed = true
	totalAfter = total + 1
	videosAfter = videos
	if m.MediaType == "video" {
		videosAfter++
	}
	return totalAfter, videosAfter, nil
}

// CommitInUploadSession — аналог CommitInSession для pin_upload_sessions.
// Берёт advisory_xact_lock по session_id, проверяет лимиты трипа, INSERT media,
// UPDATE pin_upload_sessions.last_activity_at — всё в одной транзакции.
func (r *MediaRepository) CommitInUploadSession(ctx context.Context, m *models.Media, sessionID string, maxMedia, maxVideos int) (totalAfter, videosAfter int, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", sessionID); err != nil {
		return 0, 0, err
	}
	var total, videos int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(*) FILTER (WHERE media_type = 'video') FROM media WHERE trip_id = $1", m.TripID).Scan(&total, &videos); err != nil {
		return 0, 0, err
	}
	if total >= maxMedia {
		return total, videos, ErrMediaLimitExceeded
	}
	if m.MediaType == "video" && videos >= maxVideos {
		return total, videos, ErrVideoLimitExceeded
	}
	cols := []string{"trip_id", "s3_key", "media_type", "captured_at", "battle_rating", "privacy_level"}
	vals := []interface{}{m.TripID, m.S3Key, m.MediaType, m.CapturedAt, m.BattleRating, m.PrivacyLevel}
	if m.PinID != nil {
		cols = append(cols, "pin_id")
		vals = append(vals, *m.PinID)
	}
	if m.Latitude != nil && m.Longitude != nil {
		cols = append(cols, "location")
		vals = append(vals, sq.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)", *m.Longitude, *m.Latitude))
	}
	if m.ContentHash != nil {
		cols = append(cols, "content_hash")
		vals = append(vals, *m.ContentHash)
	}
	if m.UploadedBy != nil {
		cols = append(cols, "uploaded_by")
		vals = append(vals, *m.UploadedBy)
	}
	if m.UploadSessionID != nil {
		cols = append(cols, "upload_session_id")
		vals = append(vals, *m.UploadSessionID)
	}
	q := psq.Insert("media").Columns(cols...).Values(vals...).Suffix("RETURNING id, created_at")
	sqlStr, args, qerr := q.ToSql()
	if qerr != nil {
		return 0, 0, qerr
	}
	if err = tx.QueryRowContext(ctx, sqlStr, args...).Scan(&m.ID, &m.CreatedAt); err != nil {
		return 0, 0, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE pin_upload_sessions SET last_activity_at = NOW() WHERE session_id = $1", sessionID); err != nil {
		return 0, 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	committed = true
	totalAfter = total + 1
	videosAfter = videos
	if m.MediaType == "video" {
		videosAfter++
	}
	return totalAfter, videosAfter, nil
}

func (r *MediaRepository) GetByID(id string) (*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_Y(location)::float as lat, ST_X(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, upload_session_id, created_at
		FROM media WHERE id = $1`
	var m models.Media
	var pinID, similarGroupID, contentHash, uploadSession sql.NullString
	var lat, lon sql.NullFloat64
	var capturedAt sql.NullTime
	err := r.db.QueryRow(sqlStr, id).Scan(&m.ID, &m.TripID, &pinID, &m.S3Key, &m.MediaType,
		&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &contentHash, &uploadSession, &m.CreatedAt)
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
	if uploadSession.Valid {
		m.UploadSessionID = &uploadSession.String
	}
	return &m, nil
}

func (r *MediaRepository) ListByTripID(tripID string) ([]*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_Y(location)::float as lat, ST_X(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, uploaded_by, upload_session_id, created_at
		FROM media WHERE trip_id = $1 ORDER BY captured_at ASC NULLS LAST, created_at ASC`
	rows, err := r.db.Query(sqlStr, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Media
	for rows.Next() {
		var m models.Media
		var pinID, similarGroupID, contentHash, uploadedBy, uploadSession sql.NullString
		var lat, lon sql.NullFloat64
		var capturedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.TripID, &pinID, &m.S3Key, &m.MediaType,
			&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &contentHash, &uploadedBy, &uploadSession, &m.CreatedAt); err != nil {
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
		if uploadedBy.Valid {
			m.UploadedBy = &uploadedBy.String
		}
		if uploadSession.Valid {
			m.UploadSessionID = &uploadSession.String
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

// DeleteOrphanSessionMedia — удаляет медиа трипа, которые не попали ни в один
// пин (pin_id IS NULL) и не входят в existingMediaIDs (то есть были загружены в текущей
// add-media сессии, но остались без привязки). Вызывается при Cancel/abandoned.
// Возвращает s3_keys удалённых медиа — вызывающий должен почистить S3.
func (r *MediaRepository) DeleteOrphanSessionMedia(tripID string, existingMediaIDs []string) ([]string, error) {
	cond := sq.And{
		sq.Eq{"trip_id": tripID},
		sq.Eq{"pin_id": nil},
	}
	if len(existingMediaIDs) > 0 {
		cond = append(cond, sq.NotEq{"id": existingMediaIDs})
	}
	selectQ := psq.Select("id", "s3_key").From("media").Where(cond)
	selectSQL, args, err := selectQ.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(selectSQL, args...)
	if err != nil {
		return nil, err
	}
	var ids []string
	var s3Keys []string
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
		s3Keys = append(s3Keys, key)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if _, err := psq.Delete("media").Where(sq.Eq{"id": ids}).RunWith(r.db).Exec(); err != nil {
		return nil, err
	}
	return s3Keys, nil
}

// SetSimilarGroupID помечает медиа как одну группу похожих внутри пина (один и тот же groupID = одна группа).
func (r *MediaRepository) SetSimilarGroupID(mediaIDs []string, groupID string) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	_, err := psq.Update("media").Set("similar_group_id", groupID).Where(sq.Eq{"id": mediaIDs}).RunWith(r.db).Exec()
	return err
}

// MarkNSFW выставляет privacy_level='restricted' для переданных медиа (цензура через ML).
func (r *MediaRepository) MarkNSFW(mediaIDs []string) error {
	if len(mediaIDs) == 0 {
		return nil
	}
	_, err := psq.Update("media").Set("privacy_level", "restricted").Where(sq.Eq{"id": mediaIDs}).RunWith(r.db).Exec()
	return err
}

// SetPrivacyLevel sets privacy_level on a single media (used by privacy aggregation worker).
// SQL guard: never overwrite restricted ("permanently private") with a lower level.
func (r *MediaRepository) SetPrivacyLevel(mediaID, level string) error {
	q := psq.Update("media").Set("privacy_level", level).Where(sq.Eq{"id": mediaID})
	if level != "restricted" {
		q = q.Where(sq.NotEq{"privacy_level": "restricted"})
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

// PickRandomForBattle возвращает до limit случайных медиа трипа, исключая restricted (NSFW). Для фотобатла.
func (r *MediaRepository) PickRandomForBattle(tripID string, limit int) ([]*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_Y(location)::float as lat, ST_X(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, created_at
		FROM media WHERE trip_id = $1 AND privacy_level <> 'restricted'
		ORDER BY RANDOM() LIMIT $2`
	rows, err := r.db.Query(sqlStr, tripID, limit)
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

// IncrementBattleRating атомарно увеличивает battle_rating победителя батла на 1 и возвращает новое значение.
func (r *MediaRepository) IncrementBattleRating(mediaID string) (int32, error) {
	var rating int32
	err := r.db.QueryRow(`UPDATE media SET battle_rating = battle_rating + 1 WHERE id = $1 RETURNING battle_rating`, mediaID).Scan(&rating)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, sql.ErrNoRows
		}
		return 0, err
	}
	return rating, nil
}

// ListWithPositiveBattleRating возвращает медиа трипа с battle_rating > 0, отсортированные по рейтингу DESC для "лучших воспоминаний".
func (r *MediaRepository) ListWithPositiveBattleRating(tripID string) ([]*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_Y(location)::float as lat, ST_X(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, created_at
		FROM media WHERE trip_id = $1 AND battle_rating > 0
		ORDER BY battle_rating DESC, captured_at ASC NULLS LAST, created_at ASC`
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
		ST_Y(location)::float as lat, ST_X(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, upload_session_id, created_at
		FROM media WHERE pin_id = $1 ORDER BY captured_at ASC NULLS LAST, created_at ASC`
	rows, err := r.db.Query(sqlStr, pinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Media
	for rows.Next() {
		var m models.Media
		var pinIDNull, similarGroupID, contentHash, uploadSession sql.NullString
		var lat, lon sql.NullFloat64
		var capturedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.TripID, &pinIDNull, &m.S3Key, &m.MediaType,
			&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &contentHash, &uploadSession, &m.CreatedAt); err != nil {
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
		if uploadSession.Valid {
			m.UploadSessionID = &uploadSession.String
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}

type FeedMedia struct {
	ID string
	S3Key string
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

// DeleteByPinID удаляет все media пина и возвращает их s3_keys для best-effort
// S3 cleanup. FK media.pin_id =
// ON DELETE SET NULL, поэтому полагаться на cascade нельзя — удаляем явно.
func (r *MediaRepository) DeleteByPinID(pinID string) ([]string, error) {
	rows, err := r.db.Query(`DELETE FROM media WHERE pin_id = $1 RETURNING s3_key`, pinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// ListByUploadSession возвращает media активной pin-upload-сессии
// (pin_id=NULL, upload_session_id=$1) в хронологическом порядке.
func (r *MediaRepository) ListByUploadSession(sessionID string) ([]*models.Media, error) {
	sqlStr := `SELECT id, trip_id, pin_id, s3_key, media_type,
		ST_Y(location)::float as lat, ST_X(location)::float as lon,
		captured_at, battle_rating, privacy_level, similar_group_id, content_hash, uploaded_by, upload_session_id, created_at
		FROM media WHERE upload_session_id = $1
		ORDER BY captured_at ASC NULLS LAST, created_at ASC`
	rows, err := r.db.Query(sqlStr, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.Media
	for rows.Next() {
		var m models.Media
		var pinID, similarGroupID, contentHash, uploadedBy, uploadSession sql.NullString
		var lat, lon sql.NullFloat64
		var capturedAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.TripID, &pinID, &m.S3Key, &m.MediaType,
			&lat, &lon, &capturedAt, &m.BattleRating, &m.PrivacyLevel, &similarGroupID, &contentHash, &uploadedBy, &uploadSession, &m.CreatedAt); err != nil {
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
		if uploadedBy.Valid {
			m.UploadedBy = &uploadedBy.String
		}
		if uploadSession.Valid {
			m.UploadSessionID = &uploadSession.String
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}

// DeleteOrphanByUploadSession удаляет media с pin_id=NULL,
// upload_session_id=$1 и возвращает s3_keys для S3 cleanup.
func (r *MediaRepository) DeleteOrphanByUploadSession(sessionID string) ([]string, error) {
	rows, err := r.db.Query(
		`DELETE FROM media WHERE upload_session_id = $1 AND pin_id IS NULL RETURNING s3_key`,
		sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
// TopMediaByPinIDs возвращает топ-N медиа на каждый пин (sorted by battle_rating DESC, id) одним запросом.
func (r *MediaRepository) TopMediaByPinIDs(pinIDs []string, limitPerPin int) (map[string][]*FeedMedia, error) {
	if len(pinIDs) == 0 || limitPerPin <= 0 {
		return nil, nil
	}
	placeholders := make([]string, len(pinIDs))
	args := make([]interface{}, len(pinIDs)+1)
	for i, id := range pinIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(pinIDs)] = limitPerPin
	sqlStr := `SELECT id, pin_id, s3_key, media_type FROM (
		SELECT id, pin_id, s3_key, media_type,
			ROW_NUMBER() OVER (PARTITION BY pin_id ORDER BY battle_rating DESC, id) AS rn
		FROM media WHERE pin_id IN (` + strings.Join(placeholders, ",") + `)
	) sub WHERE rn <= $` + fmt.Sprintf("%d", len(pinIDs)+1)
	rows, err := r.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]*FeedMedia)
	for rows.Next() {
		var id, pinID, s3Key, mediaType string
		if err := rows.Scan(&id, &pinID, &s3Key, &mediaType); err != nil {
			return nil, err
		}
		out[pinID] = append(out[pinID], &FeedMedia{ID: id, S3Key: s3Key, MediaType: mediaType})
	}
	return out, rows.Err()
}
