-- name: TripSettingsEnsureDefault :exec
INSERT INTO trip_settings (user_id, trip_id, notifications_enabled) VALUES ($1, $2, true)
ON CONFLICT (user_id, trip_id) DO NOTHING;

-- name: TripSettingsUpdateNotifications :execrows
UPDATE trip_settings SET notifications_enabled = $1 WHERE trip_id = $2 AND user_id = $3;

-- name: TripSettingsGetByTripAndUsers :many
SELECT user_id, notifications_enabled
FROM trip_settings
WHERE trip_id = $1 AND user_id = ANY($2::uuid[]);
