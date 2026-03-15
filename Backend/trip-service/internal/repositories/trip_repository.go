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
		"likes_count", "dislikes_count", "cover_url", "is_published", "is_generated",
		"created_at", "updated_at",
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
		&t.LikesCount, &t.DislikesCount, &coverURL, &t.IsPublished, &t.IsGenerated,
		&t.CreatedAt, &t.UpdatedAt,
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
		"t.likes_count", "t.dislikes_count", "t.cover_url", "t.is_published", "t.is_generated",
		"t.created_at", "t.updated_at",
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
			&t.LikesCount, &t.DislikesCount, &coverURL, &t.IsPublished, &t.IsGenerated,
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

func (r *TripRepository) Update(t *models.Trip) error {
	u := psq.Update("trips").
		Set("updated_at", sq.Expr("NOW()")).
		Set("name", t.Name).
		Set("description", t.Description).
		Set("category", t.Category).
		Set("season", t.Season).
		Set("privacy_level", t.PrivacyLevel).
		Set("cover_url", t.CoverURL)
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
