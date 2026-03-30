-- name: AddMediaSessionCreate :one
INSERT INTO add_media_sessions (trip_id, existing_media_ids) VALUES ($1, $2)
RETURNING session_id;

-- name: AddMediaSessionGet :one
SELECT trip_id, existing_media_ids FROM add_media_sessions WHERE session_id = $1;

-- name: AddMediaSessionExists :one
SELECT COUNT(1)::int AS n FROM add_media_sessions WHERE trip_id = $1 AND session_id = $2;
