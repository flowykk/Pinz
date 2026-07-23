package repositories

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"pinz/backend/statistics-service/internal/models"
)

// TripLocationsRepository — факты «трип T содержит локацию L» (ER: TRIP_LOCATIONS).
// Агрегируется по списку trip_ids, который API Gateway получает из trip-service
// для текущего пользователя (поддержка поздних участников — их trips автоматически
// попадают в агрегат без backfill).
type TripLocationsRepository struct {
	db *sql.DB
}

func NewTripLocationsRepository(db *sql.DB) *TripLocationsRepository {
	return &TripLocationsRepository{db: db}
}

// Upsert записывает факт «трип T имеет локацию L». Идемпотентно (ON CONFLICT DO NOTHING).
func (r *TripLocationsRepository) Upsert(ctx context.Context, tripID string, locationID int32) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_locations (trip_id, location_id, recorded_at)
		VALUES ($1::uuid, $2, NOW())
		ON CONFLICT (trip_id, location_id) DO NOTHING`, tripID, locationID)
	return err
}

// DeleteByTripID — зачистить записи трипа при TRIP_DELETED.
func (r *TripLocationsRepository) DeleteByTripID(ctx context.Context, tripID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM trip_locations WHERE trip_id = $1`, tripID)
	return err
}

// AggregateVisitedByTripIDs — ключевой read-запрос. Возвращает список локаций
// с visit_count = COUNT(DISTINCT trip_id), отфильтрованный по опциональному типу.
func (r *TripLocationsRepository) AggregateVisitedByTripIDs(ctx context.Context, tripIDs []string, typeFilter string) ([]*models.VisitedLocation, error) {
	if len(tripIDs) == 0 {
		return []*models.VisitedLocation{}, nil
	}
	query := `
		SELECT g.id, g.name, g.type, COALESCE(g.parent_id, 0),
		 COUNT(DISTINCT tl.trip_id)::int,
		 MAX(tl.recorded_at)
		FROM trip_locations tl
		JOIN geo_registry g ON g.id = tl.location_id
		WHERE tl.trip_id = ANY($1::uuid[])`
	args := []interface{}{pq.Array(tripIDs)}
	if typeFilter != "" {
		query += ` AND g.type = $2`
		args = append(args, typeFilter)
	}
	query += `
		GROUP BY g.id, g.name, g.type, g.parent_id
		ORDER BY MAX(tl.recorded_at) DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*models.VisitedLocation, 0)
	for rows.Next() {
		var v models.VisitedLocation
		if err := rows.Scan(&v.LocationID, &v.Name, &v.Type, &v.ParentID, &v.VisitCount, &v.LastVisitAt); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}
