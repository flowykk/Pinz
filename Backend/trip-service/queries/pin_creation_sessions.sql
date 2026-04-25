-- name: PinCreationSessionCreate :one
-- ON CONFLICT DO NOTHING + UNIQUE-индекс idx_pin_creation_sessions_active_per_trip:
-- если активная сессия для трипа уже есть, возвращает sql.ErrNoRows — сервис
-- интерпретирует это как ErrPinCreationSessionActive.
INSERT INTO pin_creation_sessions (trip_id, initiator_user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
RETURNING session_id;

-- name: PinCreationSessionGet :one
SELECT session_id, trip_id, initiator_user_id, draft_snapshot,
 created_at, last_activity_at, closed_at, close_reason
FROM pin_creation_sessions
WHERE session_id = $1;

-- name: PinCreationSessionGetActiveForTrip :one
SELECT session_id, trip_id, initiator_user_id, draft_snapshot,
 created_at, last_activity_at, closed_at, close_reason
FROM pin_creation_sessions
WHERE trip_id = $1 AND closed_at IS NULL;

-- name: PinCreationSessionTouch :exec
UPDATE pin_creation_sessions
SET last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL;

-- name: PinCreationSessionSetSnapshot :exec
UPDATE pin_creation_sessions
SET draft_snapshot = $2, last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL;

-- name: PinCreationSessionClose :one
-- RETURNING чтобы сервис различал "уже закрыта" и "не существует".
UPDATE pin_creation_sessions
SET closed_at = NOW(), close_reason = $2, last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL
RETURNING session_id;
