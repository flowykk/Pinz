-- name: DeviceTokenUpsert :one
-- UPSERT: если apns_token уже есть — переносим на нового user_id и обновляем updated_at.
INSERT INTO device_tokens (user_id, apns_token)
VALUES ($1, $2)
ON CONFLICT (apns_token) DO UPDATE
 SET user_id = EXCLUDED.user_id,
 updated_at = NOW()
RETURNING id;

-- name: DeviceTokenDelete :execrows
DELETE FROM device_tokens WHERE user_id = $1 AND apns_token = $2;

-- name: DeviceTokenListByUser :many
SELECT id, user_id, apns_token, updated_at
FROM device_tokens
WHERE user_id = $1;

-- name: DeviceTokenListByUsers :many
SELECT id, user_id, apns_token, updated_at
FROM device_tokens
WHERE user_id = ANY($1::uuid[]);

-- name: DeviceTokenDeleteByToken :execrows
DELETE FROM device_tokens WHERE apns_token = $1;
