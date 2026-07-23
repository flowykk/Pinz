-- +goose Up
DROP INDEX IF EXISTS idx_media_pin_creation_session;
DROP INDEX IF EXISTS idx_media_pin_addition_session;
ALTER TABLE media DROP COLUMN IF EXISTS pin_creation_session_id;
ALTER TABLE media DROP COLUMN IF EXISTS pin_addition_session_id;
DROP TABLE IF EXISTS pin_creation_sessions;
DROP TABLE IF EXISTS pin_media_addition_sessions;

-- +goose Down
-- no-op: данных нет.
