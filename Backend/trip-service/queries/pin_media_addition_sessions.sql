-- name: PinMediaAdditionSessionCreate :one
-- ON CONFLICT DO NOTHING: если активная сессия для пина уже есть, UNIQUE-индекс
-- idx_pin_media_addition_sessions_active_per_pin сработает и Create вернёт
-- sql.ErrNoRows — сервис интерпретирует это как «другая сессия уже идёт».
INSERT INTO pin_media_addition_sessions (trip_id, pin_id, initiator_user_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
RETURNING session_id;

-- name: PinMediaAdditionSessionGet :one
SELECT session_id, trip_id, pin_id, initiator_user_id, draft_snapshot,
 created_at, last_activity_at, closed_at, close_reason
FROM pin_media_addition_sessions
WHERE session_id = $1;

-- name: PinMediaAdditionSessionGetActiveForPin :one
SELECT session_id, trip_id, pin_id, initiator_user_id, draft_snapshot,
 created_at, last_activity_at, closed_at, close_reason
FROM pin_media_addition_sessions
WHERE pin_id = $1 AND closed_at IS NULL;

-- name: PinMediaAdditionSessionTouch :exec
UPDATE pin_media_addition_sessions
SET last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL;

-- name: PinMediaAdditionSessionSetSnapshot :exec
UPDATE pin_media_addition_sessions
SET draft_snapshot = $2, last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL;

-- name: PinMediaAdditionSessionClose :one
-- RETURNING чтобы сервис знал, не была ли сессия уже закрыта (idempotent close).
UPDATE pin_media_addition_sessions
SET closed_at = NOW(), close_reason = $2, last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL
RETURNING session_id;
