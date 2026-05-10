package repositories

import (
	"context"
	"database/sql"
	"errors"

	"pinz/backend/statistics-service/internal/models"
)

// GeoRegistryRepository работает со справочником гео-объектов (ER: GEO_REGISTRY).
// Заполняется из событий TRIP_LOCATIONS_ADDED, владеет реестром на уровне statistics-service.
type GeoRegistryRepository struct {
	db *sql.DB
}

func NewGeoRegistryRepository(db *sql.DB) *GeoRegistryRepository {
	return &GeoRegistryRepository{db: db}
}

func (r *GeoRegistryRepository) Upsert(ctx context.Context, loc *models.GeoLocation) error {
	if loc == nil || loc.ID == 0 {
		return errors.New("geo_registry: invalid location")
	}
	var parent interface{}
	if loc.ParentID != nil {
		parent = *loc.ParentID
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO geo_registry (id, parent_id, name, type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			parent_id = EXCLUDED.parent_id,
			name = EXCLUDED.name,
			type = EXCLUDED.type`,
		loc.ID, parent, loc.Name, loc.Type)
	return err
}

// EnsureByName идемпотентно создаёт/находит записи страны и города.
// Возвращает master geo_registry строки (с уже выставленным id) — они
// уезжают в payload PIN_LOCATIONS_RESOLVED для mirror в trip-service.
// Любой из аргументов может быть пустым: тогда соответствующая возвращаемая
// строка nil.
func (r *GeoRegistryRepository) EnsureByName(ctx context.Context, countryName, cityName string) (*models.GeoLocation, *models.GeoLocation, error) {
	if countryName == "" && cityName == "" {
		return nil, nil, nil
	}

	var country *models.GeoLocation
	if countryName != "" {
		row, err := r.upsertByName(ctx, countryName, "country", nil)
		if err != nil {
			return nil, nil, err
		}
		country = row
	}

	var city *models.GeoLocation
	if cityName != "" {
		var parentID *int32
		if country != nil {
			parentID = &country.ID
		}
		row, err := r.upsertByName(ctx, cityName, "city", parentID)
		if err != nil {
			return country, nil, err
		}
		city = row
	}
	return country, city, nil
}

// upsertByName реализует идемпотентный INSERT … ON CONFLICT по уникальному
// индексу (name, type) для стран (parent_id IS NULL) или (name, type, parent_id)
// для городов. RETURNING + сабселект страхует от случая, когда ON CONFLICT
// не вернёт строку (parent_id NULL vs NOT NULL по разным индексам).
func (r *GeoRegistryRepository) upsertByName(ctx context.Context, name, kind string, parentID *int32) (*models.GeoLocation, error) {
	var (
		row    models.GeoLocation
		parent sql.NullInt32
	)

	if parentID == nil {
		err := r.db.QueryRowContext(ctx, `
			WITH ins AS (
			 INSERT INTO geo_registry (name, type)
			 VALUES ($1, $2)
			 ON CONFLICT (name, type) WHERE parent_id IS NULL DO NOTHING
			 RETURNING id, parent_id, name, type
			)
			SELECT id, parent_id, name, type FROM ins
			UNION ALL
			SELECT id, parent_id, name, type FROM geo_registry
			 WHERE name = $1 AND type = $2 AND parent_id IS NULL
			LIMIT 1`, name, kind).Scan(&row.ID, &parent, &row.Name, &row.Type)
		if err != nil {
			return nil, err
		}
	} else {
		err := r.db.QueryRowContext(ctx, `
			WITH ins AS (
			 INSERT INTO geo_registry (name, type, parent_id)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (name, type, parent_id) WHERE parent_id IS NOT NULL DO NOTHING
			 RETURNING id, parent_id, name, type
			)
			SELECT id, parent_id, name, type FROM ins
			UNION ALL
			SELECT id, parent_id, name, type FROM geo_registry
			 WHERE name = $1 AND type = $2 AND parent_id = $3
			LIMIT 1`, name, kind, *parentID).Scan(&row.ID, &parent, &row.Name, &row.Type)
		if err != nil {
			return nil, err
		}
	}
	if parent.Valid {
		v := parent.Int32
		row.ParentID = &v
	}
	return &row, nil
}

func (r *GeoRegistryRepository) GetByID(ctx context.Context, id int32) (*models.GeoLocation, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parent_id, name, type FROM geo_registry WHERE id = $1`, id)
	var loc models.GeoLocation
	var parent sql.NullInt32
	if err := row.Scan(&loc.ID, &parent, &loc.Name, &loc.Type); err != nil {
		return nil, err
	}
	if parent.Valid {
		v := parent.Int32
		loc.ParentID = &v
	}
	return &loc, nil
}
