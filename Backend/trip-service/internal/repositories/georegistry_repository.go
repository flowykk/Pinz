package repositories

// TODO: Trip Service временно владеет записью в geo_registry, т.к. Statistics Service ещё не реализован.
// Целевое решение (vkr.txt 2.5.4): Statistics Service — единственный владелец GEO_REGISTRY,
// Trip Service хранит read-only реплику. При реализации statistics-service:
// 1. Перенести вызов BigDataCloud и INSERT в geo_registry в statistics-service.
// 2. Настроить синхронизацию geo_registry из statistics-service → trip-service (CDC/events).
// 3. Убрать Upsert-методы из trip-service, оставить только read-запросы.

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/db/sqlcdb"
)

// GeoRegistryRepository работает с локальной репликой GEO_REGISTRY и связью TRIP_LOCATIONS.
type GeoRegistryRepository struct {
	q *sqlcdb.Queries
}

func NewGeoRegistryRepository(db *sql.DB) *GeoRegistryRepository {
	return &GeoRegistryRepository{q: sqlcdb.New(db)}
}

func (r *GeoRegistryRepository) EnsureLocationByName(ctx context.Context, countryName, cityName string) (countryID, cityID *int, displayName string, err error) {
	if countryName == "" && cityName == "" {
		return nil, nil, "", nil
	}

	var cID *int
	if countryName != "" {
		// Сначала ищем существующую запись; если нет — вставляем (upsert).
		id, err := r.q.GeoRegistryFindCountryByName(ctx, countryName)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, nil, "", err
			}
			// Не найдено — вставляем.
			id, err = r.q.GeoRegistryUpsertCountry(ctx, countryName)
			if err != nil {
				return nil, nil, "", err
			}
		}
		v := int(id)
		cID = &v
	}

	var cityIDPtr *int
	if cityName != "" {
		var id int32
		var err error
		if cID != nil {
			id, err = r.q.GeoRegistryFindCityByNameAndParent(ctx, sqlcdb.GeoRegistryFindCityByNameAndParentParams{
				Name:     cityName,
				ParentID: sql.NullInt32{Int32: int32(*cID), Valid: true},
			})
		} else {
			id, err = r.q.GeoRegistryFindCityByNameNoParent(ctx, cityName)
		}
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, nil, "", err
			}
			// Не найдено — вставляем.
			parentID := sql.NullInt32{}
			if cID != nil {
				parentID = sql.NullInt32{Int32: int32(*cID), Valid: true}
			}
			id, err = r.q.GeoRegistryUpsertCity(ctx, sqlcdb.GeoRegistryUpsertCityParams{
				Name:     cityName,
				ParentID: parentID,
			})
			if err != nil {
				return nil, nil, "", err
			}
		}
		v := int(id)
		cityIDPtr = &v
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

// UpsertTripLocations заполняет TRIP_LOCATIONS для трипа по списку locationID (страна/город).
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
