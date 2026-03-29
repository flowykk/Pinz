-- name: GeoRegistryFindCountryByName :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'Country' LIMIT 1;

-- name: GeoRegistryFindCityByNameAndParent :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'City' AND parent_id = $2 LIMIT 1;

-- name: GeoRegistryFindCityByNameNoParent :one
SELECT id FROM geo_registry WHERE name = $1 AND type = 'City' LIMIT 1;

-- name: GeoRegistryFindIDsByNamePattern :many
SELECT id FROM geo_registry
WHERE name ILIKE $1 AND (type = 'Country' OR type = 'City');

-- name: TripLocationInsert :exec
INSERT INTO trip_locations (trip_id, location_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;
