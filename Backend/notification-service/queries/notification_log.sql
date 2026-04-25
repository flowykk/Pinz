-- name: NotificationLogInsert :execrows
-- Записывает факт отправки пуша. Уникальный ключ (event_id, apns_token)
-- обеспечивает идемпотентность: повторная попытка получит 0 rows.
INSERT INTO notification_log (event_id, apns_token)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: NotificationLogExists :one
SELECT EXISTS(
 SELECT 1 FROM notification_log WHERE event_id = $1 AND apns_token = $2
);
