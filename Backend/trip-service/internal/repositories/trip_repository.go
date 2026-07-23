package repositories

import (
	"database/sql"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"

	"pinz/backend/trip-service/internal/models"
)

var psq = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type TripRepository struct {
	db *sql.DB
}

func NewTripRepository(db *sql.DB) *TripRepository {
	return &TripRepository{db: db}
}

func (r *TripRepository) Create(t *models.Trip) error {
	q := psq.Insert("trips").
		Columns(
			"owner_user_id", "name", "description", "category", "season",
			"status", "privacy_level", "start_date", "end_date",
			"likes_count", "dislikes_count", "cover_url", "is_published", "is_generated",
		).
		Values(
			t.OwnerUserID, t.Name, t.Description, t.Category, t.Season,
			t.Status, t.PrivacyLevel, t.StartDate, t.EndDate,
			t.LikesCount, t.DislikesCount, t.CoverURL, t.IsPublished, t.IsGenerated,
		).
		Suffix("RETURNING id, created_at, updated_at")
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return err
	}
	var id string
	var createdAt, updatedAt interface{}
	err = r.db.QueryRow(sqlStr, args...).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return err
	}
	t.ID = id
	return nil
}

func (r *TripRepository) GetByID(id string) (*models.Trip, error) {
	q := psq.Select(
		"id", "owner_user_id", "name", "description", "category", "season",
		"status", "privacy_level", "start_date", "end_date",
		"likes_count", "dislikes_count", "cover_url", "is_published", "is_generated", "is_soft_deleted",
		"name_censored", "description_censored",
		"created_at", "updated_at",
		"(SELECT COUNT(*) FROM media m WHERE m.trip_id = trips.id)",
		"(SELECT COUNT(*) FROM trip_participants tp WHERE tp.trip_id = trips.id)",
		"(SELECT COUNT(*) FROM pins p WHERE p.trip_id = trips.id)",
	).From("trips").Where(sq.Eq{"id": id})
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRow(sqlStr, args...)
	var t models.Trip
	var desc, coverURL sql.NullString
	var startDate, endDate sql.NullTime
	err = row.Scan(
		&t.ID, &t.OwnerUserID, &t.Name, &desc, &t.Category, &t.Season,
		&t.Status, &t.PrivacyLevel, &startDate, &endDate,
		&t.LikesCount, &t.DislikesCount, &coverURL, &t.IsPublished, &t.IsGenerated, &t.IsSoftDeleted,
		&t.NameCensored, &t.DescriptionCensored,
		&t.CreatedAt, &t.UpdatedAt,
		&t.MediaCount, &t.ParticipantsCount, &t.PinsCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	if desc.Valid {
		t.Description = desc.String
	}
	if coverURL.Valid {
		t.CoverURL = coverURL.String
	}
	if startDate.Valid {
		t.StartDate = &startDate.Time
	}
	if endDate.Valid {
		t.EndDate = &endDate.Time
	}
	return &t, nil
}

func (r *TripRepository) ListByUserID(userID string, limit, offset int32) ([]*models.Trip, error) {
	// Trips where user is participant (join with trip_participants).
	q := psq.Select(
		"t.id", "t.owner_user_id", "t.name", "t.description", "t.category", "t.season",
		"t.status", "t.privacy_level", "t.start_date", "t.end_date",
		"t.likes_count", "t.dislikes_count", "t.cover_url", "t.is_published", "t.is_generated", "t.is_soft_deleted",
		"t.name_censored", "t.description_censored",
		"t.created_at", "t.updated_at",
		"(SELECT COUNT(*) FROM media m WHERE m.trip_id = t.id)",
		"(SELECT COUNT(*) FROM trip_participants tp2 WHERE tp2.trip_id = t.id)",
		"(SELECT COUNT(*) FROM pins p WHERE p.trip_id = t.id)",
	).From("trips t").
		InnerJoin("trip_participants tp ON tp.trip_id = t.id").
		Where(sq.Eq{"tp.user_id": userID}).
		Where(sq.Eq{"t.is_generated": false}).
		Where(sq.Eq{"t.is_soft_deleted": false}).
		OrderBy("t.updated_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset))
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trips []*models.Trip
	for rows.Next() {
		var t models.Trip
		var desc, coverURL sql.NullString
		var startDate, endDate sql.NullTime
		if err := rows.Scan(
			&t.ID, &t.OwnerUserID, &t.Name, &desc, &t.Category, &t.Season,
			&t.Status, &t.PrivacyLevel, &startDate, &endDate,
			&t.LikesCount, &t.DislikesCount, &coverURL, &t.IsPublished, &t.IsGenerated, &t.IsSoftDeleted,
			&t.NameCensored, &t.DescriptionCensored,
			&t.CreatedAt, &t.UpdatedAt,
			&t.MediaCount, &t.ParticipantsCount, &t.PinsCount,
		); err != nil {
			return nil, err
		}
		if desc.Valid {
			t.Description = desc.String
		}
		if coverURL.Valid {
			t.CoverURL = coverURL.String
		}
		if startDate.Valid {
			t.StartDate = &startDate.Time
		}
		if endDate.Valid {
			t.EndDate = &endDate.Time
		}
		trips = append(trips, &t)
	}
	return trips, rows.Err()
}

func (r *TripRepository) Update(t *models.Trip) error {
	u := psq.Update("trips").
		Set("updated_at", sq.Expr("NOW()")).
		Set("name", t.Name).
		Set("description", t.Description).
		Set("category", t.Category).
		Set("season", t.Season).
		Set("privacy_level", t.PrivacyLevel).
		Set("cover_url", t.CoverURL).
		Set("is_published", t.IsPublished)
	if t.StartDate != nil {
		u = u.Set("start_date", t.StartDate)
	} else {
		u = u.Set("start_date", nil)
	}
	if t.EndDate != nil {
		u = u.Set("end_date", t.EndDate)
	} else {
		u = u.Set("end_date", nil)
	}
	u = u.Where(sq.Eq{"id": t.ID})
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

func (r *TripRepository) Delete(id string) error {
	res, err := psq.Delete("trips").Where(sq.Eq{"id": id}).RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateCoverURL sets cover_url (may be empty to clear) and bumps updated_at.
func (r *TripRepository) UpdateCoverURL(tripID, s3Key string) error {
	q := psq.Update("trips").
		Set("updated_at", sq.Expr("NOW()")).
		Set("cover_url", s3Key).
		Where(sq.Eq{"id": tripID})
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

// SetPrivacyLevel updates only trip privacy_level (used by privacy aggregation worker).
// SQL guard: never overwrite restricted ("permanently private") with a lower level.
func (r *TripRepository) SetPrivacyLevel(tripID, level string) error {
	q := psq.Update("trips").Set("privacy_level", level).Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": tripID})
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

// nil pointer = поле не обновлять.
func (r *TripRepository) SetTextCensored(tripID string, nameCensored, descriptionCensored *bool) error {
	if nameCensored == nil && descriptionCensored == nil {
		return nil
	}
	q := psq.Update("trips").Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": tripID})
	if nameCensored != nil {
		q = q.Set("name_censored", *nameCensored)
	}
	if descriptionCensored != nil {
		q = q.Set("description_censored", *descriptionCensored)
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

// SetStatus updates only trip status (for flow: UPLOADING, DRAFT_GROUPING_REVIEW, PROCESSING, DRAFT_FINAL_REVIEW, READY).
func (r *TripRepository) SetStatus(tripID, status string) error {
	res, err := psq.Update("trips").Set("status", status).Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": tripID}).RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SetSoftDeleted sets is_soft_deleted=true.
func (r *TripRepository) ListAbandonedGenerated(minAge time.Duration, limit int) ([]string, error) {
	const sqlStr = `
SELECT t.id FROM trips t
WHERE t.is_generated = true
  AND t.is_soft_deleted = false
  AND t.created_at < NOW() - make_interval(secs => $1::float8)
  AND NOT EXISTS (
    SELECT 1 FROM favourite f
    WHERE f.user_id = t.owner_user_id AND f.trip_id = t.id
  )
ORDER BY t.created_at ASC
LIMIT $2`
	rows, err := r.db.Query(sqlStr, minAge.Seconds(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *TripRepository) SetSoftDeleted(tripID string) error {
	res, err := psq.Update("trips").Set("is_soft_deleted", true).Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": tripID}).RunWith(r.db).Exec()
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListFeed returns published trips for the feed. Excludes soft-deleted. Optional filters: category, season, locationIDs. sortBy: "date" or "rating".
func (r *TripRepository) ListFeed(limit, offset int32, category, season string, locationIDs []int, sortBy string) ([]*models.Trip, error) {
	q := psq.Select(
		"t.id", "t.owner_user_id", "t.name", "t.description", "t.category", "t.season",
		"t.status", "t.privacy_level", "t.start_date", "t.end_date",
		"t.likes_count", "t.dislikes_count", "t.cover_url", "t.is_published", "t.is_generated", "t.is_soft_deleted",
		"t.name_censored", "t.description_censored",
		"t.created_at", "t.updated_at",
	).From("trips t").
		Where(sq.Eq{"t.is_published": true}).
		Where(sq.Eq{"t.is_soft_deleted": false}).
		Where(sq.Eq{"t.privacy_level": "public"})
	if category != "" {
		q = q.Where(sq.Eq{"t.category": category})
	}
	if season != "" {
		q = q.Where(sq.Eq{"t.season": season})
	}
	if len(locationIDs) > 0 {
		q = q.InnerJoin("trip_locations tl ON tl.trip_id = t.id").Where(sq.Eq{"tl.location_id": locationIDs})
	}
	switch sortBy {
	case "rating":
		q = q.OrderBy("(t.likes_count - t.dislikes_count) DESC", "t.updated_at DESC")
	default:
		q = q.OrderBy("t.updated_at DESC")
	}
	q = q.Limit(uint64(limit)).Offset(uint64(offset))
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trips []*models.Trip
	for rows.Next() {
		var t models.Trip
		var desc, coverURL sql.NullString
		var startDate, endDate sql.NullTime
		if err := rows.Scan(
			&t.ID, &t.OwnerUserID, &t.Name, &desc, &t.Category, &t.Season,
			&t.Status, &t.PrivacyLevel, &startDate, &endDate,
			&t.LikesCount, &t.DislikesCount, &coverURL, &t.IsPublished, &t.IsGenerated, &t.IsSoftDeleted,
			&t.NameCensored, &t.DescriptionCensored,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if desc.Valid {
			t.Description = desc.String
		}
		if coverURL.Valid {
			t.CoverURL = coverURL.String
		}
		if startDate.Valid {
			t.StartDate = &startDate.Time
		}
		if endDate.Valid {
			t.EndDate = &endDate.Time
		}
		trips = append(trips, &t)
	}
	return trips, rows.Err()
}

// TripSummary — компактная сводка по трипу для statistics (без метаданных).
type TripSummary struct {
	TripID string
	PinsCount int32
	MediaCount int32
}

// ListSummariesByUserID возвращает все трипы пользователя (без пагинации) с count'ами
// пинов и медиа. Используется API Gateway для агрегации профильной статистики.
func (r *TripRepository) ListSummariesByUserID(userID string) ([]*TripSummary, error) {
	q := psq.Select(
		"t.id",
		"(SELECT COUNT(*) FROM pins p WHERE p.trip_id = t.id)",
		"(SELECT COUNT(*) FROM media m WHERE m.trip_id = t.id)",
	).From("trips t").
		InnerJoin("trip_participants tp ON tp.trip_id = t.id").
		Where(sq.Eq{"tp.user_id": userID}).
		Where(sq.Eq{"t.is_generated": false}).
		Where(sq.Eq{"t.is_soft_deleted": false})
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*TripSummary, 0)
	for rows.Next() {
		var s TripSummary
		if err := rows.Scan(&s.TripID, &s.PinsCount, &s.MediaCount); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// NotificationTripCandidate — сводный формат для notification-service
// scheduler'а: id, name, участники, ключевые даты. YearsElapsed — целое число
// лет от start_date (для текста «N лет назад»).
type NotificationTripCandidate struct {
	TripID string
	Name string
	Participants []string
	StartDateUnix int64
	EndDateUnix int64
	YearsElapsed int32
}

// ListAnniversaryCandidates — трипы, у которых сегодня исполняется
// ровно N лет с start_date (N ∈ {1, 2, 3,...}). Сравниваем день+месяц с
// сегодняшним и возвращаем число полных лет. Трипы без start_date пропускаем.
func (r *TripRepository) ListAnniversaryCandidates(today int64) ([]*NotificationTripCandidate, error) {
	const q = `
		SELECT t.id, t.name,
			COALESCE(t.start_date, to_timestamp(0)),
			COALESCE(t.end_date, to_timestamp(0)),
			EXTRACT(YEAR FROM age(date_trunc('day', to_timestamp($1)), date_trunc('day', t.start_date)))::int AS years_elapsed,
			COALESCE(array_agg(tp.user_id) FILTER (WHERE tp.user_id IS NOT NULL), '{}')
		FROM trips t
		LEFT JOIN trip_participants tp ON tp.trip_id = t.id
		WHERE t.is_soft_deleted = false
			AND t.start_date IS NOT NULL
			AND EXTRACT(MONTH FROM t.start_date) = EXTRACT(MONTH FROM to_timestamp($1))
			AND EXTRACT(DAY FROM t.start_date) = EXTRACT(DAY FROM to_timestamp($1))
			AND date_trunc('day', t.start_date) < date_trunc('day', to_timestamp($1))
		GROUP BY t.id
	`
	return r.queryNotificationCandidates(q, today, true)
}

// ListEndedMonthAgoCandidates возвращает трипы, end_date которых пришёлся на
// today - 1 month.
func (r *TripRepository) ListEndedMonthAgoCandidates(today int64) ([]*NotificationTripCandidate, error) {
	const q = `
		SELECT t.id, t.name,
			COALESCE(t.start_date, to_timestamp(0)),
			COALESCE(t.end_date, to_timestamp(0)),
			COALESCE(array_agg(tp.user_id) FILTER (WHERE tp.user_id IS NOT NULL), '{}')
		FROM trips t
		LEFT JOIN trip_participants tp ON tp.trip_id = t.id
		WHERE t.is_soft_deleted = false
			AND t.end_date IS NOT NULL
			AND date_trunc('day', t.end_date) = date_trunc('day', to_timestamp($1) - interval '1 month')
		GROUP BY t.id
	`
	return r.queryNotificationCandidates(q, today, false)
}

func (r *TripRepository) queryNotificationCandidates(query string, today int64, withYears bool) ([]*NotificationTripCandidate, error) {
	rows, err := r.db.Query(query, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*NotificationTripCandidate, 0)
	for rows.Next() {
		var c NotificationTripCandidate
		var startDate, endDate sql.NullTime
		var participants pq.StringArray
		if withYears {
			if err := rows.Scan(&c.TripID, &c.Name, &startDate, &endDate, &c.YearsElapsed, &participants); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&c.TripID, &c.Name, &startDate, &endDate, &participants); err != nil {
				return nil, err
			}
		}
		if startDate.Valid {
			c.StartDateUnix = startDate.Time.Unix()
		}
		if endDate.Valid {
			c.EndDateUnix = endDate.Time.Unix()
		}
		c.Participants = []string(participants)
		out = append(out, &c)
	}
	return out, rows.Err()
}
