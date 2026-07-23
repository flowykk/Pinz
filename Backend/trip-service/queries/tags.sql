-- name: TagDeleteForPin :exec
DELETE FROM tags WHERE trip_id = $1 AND pin_id = $2;

-- name: TagInsert :one
INSERT INTO tags (trip_id, pin_id, tag) VALUES ($1, $2, $3) RETURNING id;

-- name: TagListByPin :many
SELECT tag FROM tags WHERE pin_id = $1;

-- name: TagListByTrip :many
SELECT pin_id, tag FROM tags WHERE trip_id = $1;

-- name: TagDeleteAllForPin :exec
DELETE FROM tags WHERE pin_id = $1;
