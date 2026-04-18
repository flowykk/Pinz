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
			name      = EXCLUDED.name,
			type      = EXCLUDED.type`,
		loc.ID, parent, loc.Name, loc.Type)
	return err
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
