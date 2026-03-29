-- name: TripPrivacyUpsert :exec
INSERT INTO trip_privacy (trip_id, user_id, privacy_level) VALUES ($1, $2, $3)
ON CONFLICT (trip_id, user_id) DO UPDATE SET privacy_level = $3;

-- name: TripPrivacyListByTrip :many
SELECT user_id, privacy_level FROM trip_privacy WHERE trip_id = $1;

-- name: PinPrivacyUpsert :exec
INSERT INTO pin_privacy (pin_id, user_id, privacy_level) VALUES ($1, $2, $3)
ON CONFLICT (pin_id, user_id) DO UPDATE SET privacy_level = $3;

-- name: PinPrivacyListByPin :many
SELECT user_id, privacy_level FROM pin_privacy WHERE pin_id = $1;

-- name: MediaPrivacyUpsert :exec
INSERT INTO media_privacy (media_id, user_id, privacy_level) VALUES ($1, $2, $3)
ON CONFLICT (media_id, user_id) DO UPDATE SET privacy_level = $3;

-- name: MediaPrivacyListByMedia :many
SELECT user_id, privacy_level FROM media_privacy WHERE media_id = $1;
