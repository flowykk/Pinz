-- name: IsEventProcessed :one
SELECT EXISTS (SELECT 1 FROM stats_event_log WHERE event_id = $1) AS processed;

-- name: MarkEventProcessed :exec
INSERT INTO stats_event_log (event_id, event_type)
VALUES ($1, $2)
ON CONFLICT (event_id) DO NOTHING;
