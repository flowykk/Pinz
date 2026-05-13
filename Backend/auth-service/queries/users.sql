-- name: GetUserByEmail :one
SELECT id, email, username, avatar_url, created_at, is_test
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
SELECT id, email, username, avatar_url, created_at, is_test
FROM users
WHERE id = $1;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE id = $1;

-- name: UpdateRefreshTokenExpiresAt :exec
UPDATE refresh_tokens SET expires_at = $2 WHERE id = $1;

-- name: DeleteUserRefreshTokens :exec
DELETE FROM refresh_tokens WHERE user_id = $1;

-- name: UpdateUsername :one
UPDATE users SET username = $2 WHERE id = $1
RETURNING id, email, username, avatar_url, created_at, is_test;

-- name: UpdateEmail :one
UPDATE users SET email = $2 WHERE id = $1
RETURNING id, email, username, avatar_url, created_at, is_test;

-- name: UpdateAvatarURL :one
UPDATE users SET avatar_url = $2 WHERE id = $1
RETURNING id, email, username, avatar_url, created_at, is_test;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetUsersByIDs :many
SELECT id, email, username, avatar_url, created_at, is_test
FROM users
WHERE id = ANY($1::uuid[]);
