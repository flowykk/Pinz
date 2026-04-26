-- name: CreateDesiredPlace :one
INSERT INTO desired_places (id, user_id, name, description, image_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, name, description, image_url, created_at;

-- name: GetDesiredPlace :one
SELECT id, user_id, name, description, image_url, created_at
FROM desired_places
WHERE id = $1;

-- name: ListDesiredPlacesByUserID :many
SELECT id, user_id, name, description, image_url, created_at
FROM desired_places
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateDesiredPlace :one
UPDATE desired_places
SET name = $2,
    description = $3,
    image_url = $4
WHERE id = $1
RETURNING id, user_id, name, description, image_url, created_at;

-- name: UpdateDesiredPlaceContent :one
UPDATE desired_places
SET name = $2,
    description = $3
WHERE id = $1
RETURNING id, user_id, name, description, image_url, created_at;

-- name: ClearDesiredPlaceImage :one
UPDATE desired_places
SET image_url = NULL
WHERE id = $1
RETURNING id, user_id, name, description, image_url, created_at;

-- name: DeleteDesiredPlace :exec
DELETE FROM desired_places WHERE id = $1;
