package repositories

// Trip-service хранит read-only реплику GEO_REGISTRY.
// Master живёт в statistics-service: stats консумит PIN_LOCATIONS_REQUESTED
// (pinz:stats:events), резолвит координаты через BigDataCloud, upsert'ит
// собственный geo_registry/trip_locations и публикует PIN_LOCATIONS_RESOLVED
// в pinz:trip:geo_events. Trip-service consumer mirror'ит строки сюда через
// MirrorByID (id приходит от master).

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

type GeoRegistryRepository struct {
	q *sqlcdb.Queries
}

func NewGeoRegistryRepository(db *sql.DB) *GeoRegistryRepository {
	return &GeoRegistryRepository{q: sqlcdb.New(db)}
}

// GeoLocation — DTO для кросс-сервисной передачи (statistics-service).
type GeoLocation struct {
	ID       int
	ParentID *int
	Name     string
	Type     string
}

// MirrorByID идемпотентно зеркалит запись master geo_registry в локальную
// реплику. id назначается master'ом (statistics-service).
func (r *GeoRegistryRepository) MirrorByID(ctx context.Context, row GeoLocation) error {
	if row.ID == 0 {
		return nil
	}
	parent := sql.NullInt32{}
	if row.ParentID != nil {
		parent = sql.NullInt32{Int32: int32(*row.ParentID), Valid: true}
	}
	return r.q.GeoRegistryMirrorByID(ctx, sqlcdb.GeoRegistryMirrorByIDParams{
		ID:       int32(row.ID),
		ParentID: parent,
		Name:     row.Name,
		Type:     row.Type,
	})
}

// FindCountryByName — точный поиск страны по имени. Возвращает sql.ErrNoRows, если не найдено.
// Используется рекомендательной системой, где регион принимается строкой.
func (r *GeoRegistryRepository) FindCountryByName(ctx context.Context, name string) (int, error) {
	id, err := r.q.GeoRegistryFindCountryByName(ctx, name)
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// FindCityByName — точный поиск города по имени без указания страны.
func (r *GeoRegistryRepository) FindCityByName(ctx context.Context, name string) (int, error) {
	id, err := r.q.GeoRegistryFindCityByNameNoParent(ctx, name)
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// FindLocationIDsByName returns geo_registry ids matching name (country or city). Case-insensitive partial match.
func (r *GeoRegistryRepository) FindLocationIDsByName(ctx context.Context, name string) ([]int, error) {
	if name == "" {
		return nil, nil
	}
	pattern := "%" + name + "%"
	ids32, err := r.q.GeoRegistryFindIDsByNamePattern(ctx, pattern)
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(ids32))
	for _, id := range ids32 {
		ids = append(ids, int(id))
	}
	return ids, nil
}

// GetLocations возвращает записи geo_registry по набору id (для DTO recommendations и fallback'ов).
func (r *GeoRegistryRepository) GetLocations(ctx context.Context, ids []int) ([]GeoLocation, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ids32 := make([]int32, 0, len(ids))
	for _, id := range ids {
		ids32 = append(ids32, int32(id))
	}
	rows, err := r.q.GeoRegistryGetByIDs(ctx, ids32)
	if err != nil {
		return nil, err
	}
	out := make([]GeoLocation, 0, len(rows))
	for _, row := range rows {
		loc := GeoLocation{ID: int(row.ID), Name: row.Name, Type: row.Type}
		if row.ParentID.Valid {
			p := int(row.ParentID.Int32)
			loc.ParentID = &p
		}
		out = append(out, loc)
	}
	return out, nil
}

func (r *GeoRegistryRepository) TripIDsAtLocation(ctx context.Context, locationID int, tripIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	if len(tripIDs) == 0 {
		return out, nil
	}
	parsed := make([]uuid.UUID, 0, len(tripIDs))
	for _, id := range tripIDs {
		u, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		parsed = append(parsed, u)
	}
	if len(parsed) == 0 {
		return out, nil
	}
	rows, err := r.q.TripLocationFilterByLocation(ctx, sqlcdb.TripLocationFilterByLocationParams{
		LocationID: int32(locationID),
		Column2:    parsed,
	})
	if err != nil {
		return nil, err
	}
	for _, id := range rows {
		out[id.String()] = struct{}{}
	}
	return out, nil
}

// UpsertTripLocations пишет связи trip↔location в локальную реплику.
// Используется как из geo consumer'а (mirror события PIN_LOCATIONS_RESOLVED),
// так и из рекомендательной системы (рекомендации по региону).
func (r *GeoRegistryRepository) UpsertTripLocations(ctx context.Context, tripID string, locationIDs []int) error {
	if tripID == "" || len(locationIDs) == 0 {
		return nil
	}
	tid, err := uuid.Parse(tripID)
	if err != nil {
		return err
	}
	for _, id := range locationIDs {
		if err := r.q.TripLocationInsert(ctx, sqlcdb.TripLocationInsertParams{
			TripID:     tid,
			LocationID: int32(id),
		}); err != nil {
			return err
		}
	}
	return nil
}
