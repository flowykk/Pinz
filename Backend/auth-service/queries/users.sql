-- name: GetUserByEmail :one
SELECT id, email, username, avatar_url, created_at
FROM users
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (id, email, username, avatar_url)
VALUES ($1, $2, $3, $4)
RETURNING created_at;

-- name: AddSession :exec
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3);

-- name: GetRefreshToken :one
SELECT id, user_id, token, expires_at
FROM refresh_tokens
WHERE token = $1;

-- name: GetUserByID :one
SELECT id, email, username, avatar_url, created_at
FROM users
WHERE id = $1;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE id = $1;

-- name: DeleteUserRefreshTokens :exec
DELETE FROM refresh_tokens WHERE user_id = $1;
