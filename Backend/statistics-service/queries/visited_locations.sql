-- name: UpsertGeoLocation :exec
INSERT INTO geo_registry (id, parent_id, name, type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
 parent_id = EXCLUDED.parent_id,
 name = EXCLUDED.name,
 type = EXCLUDED.type;

-- name: UpsertTripLocation :exec
INSERT INTO trip_locations (trip_id, location_id, recorded_at)
VALUES ($1, $2, NOW())
ON CONFLICT (trip_id, location_id) DO NOTHING;

-- name: DeleteTripLocationsByTrip :exec
DELETE FROM trip_locations WHERE trip_id = $1;

-- name: AggregateVisitedByTrips :many
SELECT g.id, g.name, g.type, COALESCE(g.parent_id, 0) AS parent_id,
 COUNT(DISTINCT m.trip_id)::int AS visit_count,
 MAX(m.recorded_at) AS last_visit_at
FROM trip_locations m
JOIN geo_registry g ON g.id = m.location_id
WHERE m.trip_id = ANY($1::uuid[])
GROUP BY g.id, g.name, g.type, g.parent_id
ORDER BY MAX(m.recorded_at) DESC;

-- name: AggregateVisitedByTripsAndType :many
SELECT g.id, g.name, g.type, COALESCE(g.parent_id, 0) AS parent_id,
 COUNT(DISTINCT m.trip_id)::int AS visit_count,
 MAX(m.recorded_at) AS last_visit_at
FROM trip_locations m
JOIN geo_registry g ON g.id = m.location_id
WHERE m.trip_id = ANY($1::uuid[]) AND g.type = $2
GROUP BY g.id, g.name, g.type, g.parent_id
ORDER BY MAX(m.recorded_at) DESC;
