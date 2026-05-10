-- name: GeoRegistryFindCountryByName :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'Country' LIMIT 1;

-- name: GeoRegistryFindCityByNameAndParent :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'City' AND parent_id = $2 LIMIT 1;

-- name: GeoRegistryFindCityByNameNoParent :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'City' LIMIT 1;

-- name: GeoRegistryFindIDsByNamePattern :many
SELECT id FROM geo_registry
WHERE name ILIKE $1 AND (type = 'Country' OR type = 'City');

-- name: GeoRegistryMirrorByID :exec
-- Зеркалит запись master geo_registry из statistics-service в локальную реплику.
-- id приходит от master, поэтому мы вставляем фиксированный id и обновляем поля.
INSERT INTO geo_registry (id, parent_id, name, type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
 parent_id = EXCLUDED.parent_id,
 name = EXCLUDED.name,
 type = EXCLUDED.type;

-- name: TripLocationInsert :exec
INSERT INTO trip_locations (trip_id, location_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GeoRegistryGetByIDs :many
SELECT id, parent_id, name, type FROM geo_registry WHERE id = ANY($1::int[]);

-- name: TripLocationFilterByLocation :many
SELECT trip_id FROM trip_locations
WHERE location_id = $1 AND trip_id = ANY($2::uuid[]);
