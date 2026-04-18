-- name: GeoRegistryFindCountryByName :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'Country' LIMIT 1;

-- name: GeoRegistryFindCityByNameAndParent :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'City' AND parent_id = $2 LIMIT 1;

-- name: GeoRegistryFindCityByNameNoParent :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'City' LIMIT 1;

-- name: GeoRegistryFindIDsByNamePattern :many
SELECT id FROM geo_registry
WHERE name ILIKE $1 AND (type = 'Country' OR type = 'City');

-- name: GeoRegistryUpsertCountry :one
INSERT INTO geo_registry (name, type)
VALUES ($1, 'Country')
ON CONFLICT (name, type) WHERE parent_id IS NULL
DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: GeoRegistryUpsertCity :one
INSERT INTO geo_registry (name, type, parent_id)
VALUES ($1, 'City', $2)
ON CONFLICT (name, type, parent_id) WHERE parent_id IS NOT NULL
DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: TripLocationInsert :exec
INSERT INTO trip_locations (trip_id, location_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GeoRegistryGetByIDs :many
SELECT id, parent_id, name, type FROM geo_registry WHERE id = ANY($1::int[]);
