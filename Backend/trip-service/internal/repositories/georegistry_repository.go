package repositories

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
)

// GeoRegistryRepository работает с локальной репликой GEO_REGISTRY и связью TRIP_LOCATIONS.
type GeoRegistryRepository struct {
	db *sql.DB
}

func NewGeoRegistryRepository(db *sql.DB) *GeoRegistryRepository {
	return &GeoRegistryRepository{db: db}
}

func (r *GeoRegistryRepository) EnsureLocationByName(ctx context.Context, countryName, cityName string) (countryID, cityID *int, displayName string, err error) {
	if countryName == "" && cityName == "" {
		return nil, nil, "", nil
	}

	// Country
	var cID *int
	if countryName != "" {
		q := psq.Select("id").From("geo_registry").Where(sq.Eq{"name": countryName, "type": "Country"}).Limit(1)
		sqlStr, args, err := q.ToSql()
		if err != nil {
			return nil, nil, "", err
		}
		var id int
		if err := r.db.QueryRowContext(ctx, sqlStr, args...).Scan(&id); err == nil {
			cID = &id
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, "", err
		}
	}

	// City
	var cityIDPtr *int
	if cityName != "" {
		builder := psq.Select("id").From("geo_registry").Where(sq.Eq{"name": cityName, "type": "City"})
		if cID != nil {
			builder = builder.Where(sq.Eq{"parent_id": *cID})
		}
		builder = builder.Limit(1)
		sqlStr, args, err := builder.ToSql()
		if err != nil {
			return nil, nil, "", err
		}
		var id int
		if err := r.db.QueryRowContext(ctx, sqlStr, args...).Scan(&id); err == nil {
			cityIDPtr = &id
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, "", err
		}
	}

	var name string
	switch {
	case cityName != "" && countryName != "":
		name = countryName + ", " + cityName
	case cityName != "":
		name = cityName
	case countryName != "":
		name = countryName
	}

	return cID, cityIDPtr, name, nil
}

// FindLocationIDsByName returns geo_registry ids matching name (country or city). Case-insensitive partial match.
func (r *GeoRegistryRepository) FindLocationIDsByName(ctx context.Context, name string) ([]int, error) {
	if name == "" {
		return nil, nil
	}
	q := psq.Select("id").From("geo_registry").
		Where(sq.ILike{"name": "%" + name + "%"}).
		Where(sq.Or{sq.Eq{"type": "Country"}, sq.Eq{"type": "City"}})
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// UpsertTripLocations заполняет TRIP_LOCATIONS для трипа по списку locationID (страна/город).
func (r *GeoRegistryRepository) UpsertTripLocations(ctx context.Context, tripID string, locationIDs []int) error {
	if tripID == "" || len(locationIDs) == 0 {
		return nil
	}
	for _, id := range locationIDs {
		// ON CONFLICT DO NOTHING эквивалент через INSERT ... ON CONFLICT в pgx, здесь — простой UPSERT через игнор ошибки уникальности.
		q := psq.Insert("trip_locations").
			Columns("trip_id", "location_id").
			Values(tripID, id).
			Suffix("ON CONFLICT DO NOTHING")
		sqlStr, args, err := q.ToSql()
		if err != nil {
			return err
		}
		if _, err := r.db.ExecContext(ctx, sqlStr, args...); err != nil {
			return err
		}
	}
	return nil
}
