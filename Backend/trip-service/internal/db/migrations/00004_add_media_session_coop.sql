-- +goose Up
-- кооперативная модель add-media сессии. Одна активная сессия на трип,
-- фигура "ведущего" на финальном ревью, часовой автоперехват, 72-часовой cleanup.

-- Закрываем все существующие сессии от — они были однопользовательские,
-- их состояние не совместимо с новой моделью. Без этого UNIQUE-индекс ниже упал бы
-- на трипах, где накопилось несколько записей add_media_sessions.
ALTER TABLE add_media_sessions ADD COLUMN current_initiator_user_id UUID;
ALTER TABLE add_media_sessions ADD COLUMN initiator_assigned_at TIMESTAMPTZ;
ALTER TABLE add_media_sessions ADD COLUMN last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE add_media_sessions ADD COLUMN closed_at TIMESTAMPTZ;
ALTER TABLE add_media_sessions ADD COLUMN close_reason TEXT;

UPDATE add_media_sessions SET closed_at = NOW(), close_reason = 'legacy' WHERE closed_at IS NULL;

CREATE UNIQUE INDEX idx_add_media_sessions_active_per_trip
 ON add_media_sessions(trip_id) WHERE closed_at IS NULL;

CREATE INDEX idx_add_media_sessions_last_activity
 ON add_media_sessions(last_activity_at) WHERE closed_at IS NULL;

-- Автор загрузки медиа. Нужен фронту в ADD_MEDIA_UPLOADING для показа «Боб загрузил N»
-- и в notification-service для исключения инициатора из рассылки.
ALTER TABLE media ADD COLUMN uploaded_by UUID;

-- +goose Down
ALTER TABLE media DROP COLUMN IF EXISTS uploaded_by;
DROP INDEX IF EXISTS idx_add_media_sessions_last_activity;
DROP INDEX IF EXISTS idx_add_media_sessions_active_per_trip;
ALTER TABLE add_media_sessions DROP COLUMN IF EXISTS close_reason;
ALTER TABLE add_media_sessions DROP COLUMN IF EXISTS closed_at;
ALTER TABLE add_media_sessions DROP COLUMN IF EXISTS last_activity_at;
ALTER TABLE add_media_sessions DROP COLUMN IF EXISTS initiator_assigned_at;
ALTER TABLE add_media_sessions DROP COLUMN IF EXISTS current_initiator_user_id;
