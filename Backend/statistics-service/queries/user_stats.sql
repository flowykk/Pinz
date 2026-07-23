-- name: GetUserStats :one
SELECT user_id, total_likes, total_dislikes, battles_finished, updated_at
FROM user_stats
WHERE user_id = $1;

-- name: IncrementTotalLikes :exec
INSERT INTO user_stats (user_id, total_likes)
VALUES ($1, GREATEST($2, 0))
ON CONFLICT (user_id) DO UPDATE SET
 total_likes = GREATEST(user_stats.total_likes + $2, 0),
 updated_at = NOW();

-- name: IncrementTotalDislikes :exec
INSERT INTO user_stats (user_id, total_dislikes)
VALUES ($1, GREATEST($2, 0))
ON CONFLICT (user_id) DO UPDATE SET
 total_dislikes = GREATEST(user_stats.total_dislikes + $2, 0),
 updated_at = NOW();

-- name: IncrementBattlesFinished :exec
INSERT INTO user_stats (user_id, battles_finished)
VALUES ($1, GREATEST($2, 0))
ON CONFLICT (user_id) DO UPDATE SET
 battles_finished = GREATEST(user_stats.battles_finished + $2, 0),
 updated_at = NOW();
