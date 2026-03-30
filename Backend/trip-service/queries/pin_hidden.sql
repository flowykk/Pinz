-- name: PinHiddenInsert :exec
INSERT INTO pin_hidden_by_user (pin_id, user_id) VALUES ($1, $2)
ON CONFLICT (pin_id, user_id) DO NOTHING;

-- name: PinHiddenListForUser :many
SELECT ph.pin_id
FROM pin_hidden_by_user ph
INNER JOIN pins p ON p.id = ph.pin_id
WHERE p.trip_id = $1 AND ph.user_id = $2;
