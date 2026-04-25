-- name: AddMediaSessionCreate :one
INSERT INTO add_media_sessions (trip_id, existing_media_ids)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
RETURNING session_id;

-- name: AddMediaSessionGet :one
SELECT trip_id, existing_media_ids FROM add_media_sessions WHERE session_id = $1;

-- name: AddMediaSessionExists :one
SELECT COUNT(1)::int AS n FROM add_media_sessions WHERE trip_id = $1 AND session_id = $2;

-- name: AddMediaSessionGetActive :one
SELECT session_id, trip_id, existing_media_ids, current_initiator_user_id, initiator_assigned_at, last_activity_at
FROM add_media_sessions
WHERE trip_id = $1 AND closed_at IS NULL;

-- name: AddMediaSessionSetInitiator :exec
UPDATE add_media_sessions
SET current_initiator_user_id = $2,
 initiator_assigned_at = $3,
 last_activity_at = $3
WHERE session_id = $1 AND closed_at IS NULL;

-- name: AddMediaSessionTouch :exec
UPDATE add_media_sessions
SET last_activity_at = $2
WHERE session_id = $1 AND closed_at IS NULL;

-- name: AddMediaSessionClose :one
UPDATE add_media_sessions
SET closed_at = $2, close_reason = $3
WHERE session_id = $1 AND closed_at IS NULL
RETURNING trip_id;

-- name: AddMediaSessionListAbandoned :many
SELECT session_id, trip_id
FROM add_media_sessions
WHERE closed_at IS NULL AND last_activity_at < $1;
