-- name: PinUploadSessionCreateForCreation :one
-- target_pin_id IS NULL: одна creation-сессия на трип. ON CONFLICT DO NOTHING +
-- partial UNIQUE → sql.ErrNoRows если активная сессия уже есть.
INSERT INTO pin_upload_sessions (trip_id, target_pin_id, initiator_user_id)
VALUES ($1, NULL, $2)
ON CONFLICT DO NOTHING
RETURNING session_id;

-- name: PinUploadSessionCreateForAddition :one
-- target_pin_id NOT NULL: одна addition-сессия на пин.
INSERT INTO pin_upload_sessions (trip_id, target_pin_id, initiator_user_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
RETURNING session_id;

-- name: PinUploadSessionGet :one
SELECT session_id, trip_id, target_pin_id, initiator_user_id, draft_snapshot, processing_status,
 created_at, last_activity_at, closed_at, close_reason
FROM pin_upload_sessions
WHERE session_id = $1;

-- name: PinUploadSessionGetActiveCreation :one
SELECT session_id, trip_id, target_pin_id, initiator_user_id, draft_snapshot, processing_status,
 created_at, last_activity_at, closed_at, close_reason
FROM pin_upload_sessions
WHERE trip_id = $1 AND target_pin_id IS NULL AND closed_at IS NULL;

-- name: PinUploadSessionGetActiveAddition :one
SELECT session_id, trip_id, target_pin_id, initiator_user_id, draft_snapshot, processing_status,
 created_at, last_activity_at, closed_at, close_reason
FROM pin_upload_sessions
WHERE target_pin_id = $1 AND closed_at IS NULL;

-- name: PinUploadSessionTouch :exec
UPDATE pin_upload_sessions
SET last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL;

-- name: PinUploadSessionSetSnapshot :exec
UPDATE pin_upload_sessions
SET draft_snapshot = $2, last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL;

-- name: PinUploadSessionSetProcessingStatus :one
-- CAS processing_status: возвращает session_id только если expected совпало.
UPDATE pin_upload_sessions
SET processing_status = sqlc.arg('next_status'), last_activity_at = NOW()
WHERE session_id = sqlc.arg('session_id')
 AND closed_at IS NULL
 AND processing_status = sqlc.arg('expected_status')
RETURNING session_id;

-- name: PinUploadSessionClose :one
UPDATE pin_upload_sessions
SET closed_at = NOW(), close_reason = $2, last_activity_at = NOW()
WHERE session_id = $1 AND closed_at IS NULL
RETURNING session_id;

-- name: PinUploadSessionListAbandoned :many
-- сессии без активности дольше threshold для cron-cleanup.
SELECT session_id, trip_id, target_pin_id, initiator_user_id
FROM pin_upload_sessions
WHERE closed_at IS NULL AND last_activity_at < $1;

-- name: PinUploadSessionDeleteClosedOlderThan :execrows
-- физическое удаление закрытых сессий старше threshold (cron-janitor).
DELETE FROM pin_upload_sessions
WHERE closed_at IS NOT NULL AND closed_at < $1;
