-- name: InvitationLinkInsert :exec
INSERT INTO invitation_links (id, trip_id, token, expires_at) VALUES ($1, $2, $3, $4);

-- name: InvitationLinkByToken :one
SELECT id, trip_id, token, expires_at, created_at FROM invitation_links WHERE token = $1;
