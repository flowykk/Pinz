-- name: FavouriteAdd :exec
INSERT INTO favourite (user_id, trip_id) VALUES ($1, $2)
ON CONFLICT (user_id, trip_id) DO NOTHING;

-- name: FavouriteRemove :execrows
DELETE FROM favourite WHERE user_id = $1 AND trip_id = $2;

-- name: FavouriteHas :one
SELECT 1 AS ok FROM favourite WHERE user_id = $1 AND trip_id = $2 LIMIT 1;

-- name: FavouriteHasByOtherUsers :one
SELECT 1 AS ok FROM favourite WHERE trip_id = $1 AND user_id != $2 LIMIT 1;

-- name: FavouriteListTripIDsByUser :many
SELECT trip_id FROM favourite WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: FavouriteListTripIDsByUserAndTrips :many
SELECT trip_id
FROM favourite
WHERE user_id = $1 AND trip_id = ANY($2::uuid[]);
