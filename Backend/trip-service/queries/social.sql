-- name: SocialGetReaction :one
SELECT reaction FROM social WHERE user_id = $1 AND trip_id = $2;

-- name: SocialGetReactionsByUserAndTrips :many
SELECT trip_id, reaction
FROM social
WHERE user_id = $1 AND trip_id = ANY($2::uuid[]);

-- name: SocialUpsert :exec
INSERT INTO social (user_id, trip_id, reaction) VALUES ($1, $2, $3)
ON CONFLICT (user_id, trip_id) DO UPDATE SET reaction = $3;

-- name: TripDecrementLikes :exec
UPDATE trips SET likes_count = GREATEST(0, likes_count - 1), updated_at = NOW() WHERE id = $1;

-- name: TripDecrementDislikes :exec
UPDATE trips SET dislikes_count = GREATEST(0, dislikes_count - 1), updated_at = NOW() WHERE id = $1;

-- name: TripIncrementLikes :exec
UPDATE trips SET likes_count = likes_count + 1, updated_at = NOW() WHERE id = $1;

-- name: TripIncrementDislikes :exec
UPDATE trips SET dislikes_count = dislikes_count + 1, updated_at = NOW() WHERE id = $1;
