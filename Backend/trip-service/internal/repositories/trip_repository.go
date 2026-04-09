package repositories

import (
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"

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
		"created_at", "updated_at",
		"(SELECT COUNT(*) FROM media m WHERE m.trip_id = trips.id)",
		"(SELECT COUNT(*) FROM trip_participants tp WHERE tp.trip_id = trips.id)",
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
		&t.CreatedAt, &t.UpdatedAt,
		&t.MediaCount, &t.ParticipantsCount,
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
		"t.created_at", "t.updated_at",
		"(SELECT COUNT(*) FROM media m WHERE m.trip_id = t.id)",
		"(SELECT COUNT(*) FROM trip_participants tp2 WHERE tp2.trip_id = t.id)",
	).From("trips t").
		InnerJoin("trip_participants tp ON tp.trip_id = t.id").
		Where(sq.Eq{"tp.user_id": userID}).
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
			&t.CreatedAt, &t.UpdatedAt,
			&t.MediaCount, &t.ParticipantsCount,
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

// SetPrivacyLevel updates only trip privacy_level (used by privacy aggregation worker).
func (r *TripRepository) SetPrivacyLevel(tripID, level string) error {
	res, err := psq.Update("trips").Set("privacy_level", level).Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": tripID}).RunWith(r.db).Exec()
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
		"t.created_at", "t.updated_at",
	).From("trips t").
		Where(sq.Eq{"t.is_published": true}).
		Where(sq.Eq{"t.is_soft_deleted": false}).
		Where(sq.Eq{"t.privacy_level": "Public"})
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
