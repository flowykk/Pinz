-- name: TripParticipantAdd :exec
INSERT INTO trip_participants (trip_id, user_id, is_admin) VALUES ($1, $2, $3);

-- name: TripParticipantListByTrip :many
SELECT trip_id, user_id, is_admin, joined_at FROM trip_participants WHERE trip_id = $1;

-- name: TripParticipantIsParticipant :one
SELECT 1 AS ok FROM trip_participants WHERE trip_id = $1 AND user_id = $2 LIMIT 1;

-- name: TripParticipantIsAdmin :one
SELECT is_admin FROM trip_participants WHERE trip_id = $1 AND user_id = $2;

-- name: TripParticipantRemove :execrows
DELETE FROM trip_participants WHERE trip_id = $1 AND user_id = $2;

-- name: TripParticipantRemoveAllByTrip :exec
DELETE FROM trip_participants WHERE trip_id = $1;

-- name: TripParticipantClearAdmin :exec
UPDATE trip_participants SET is_admin = false WHERE trip_id = $1;

-- name: TripParticipantSetAdmin :exec
UPDATE trip_participants SET is_admin = true WHERE trip_id = $1 AND user_id = $2;
