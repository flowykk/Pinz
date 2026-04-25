-- +goose Up
-- ТЗ 4.1, 4.6-4.11: создание одиночного пина в READY-трипе через sessioned флоу
-- с ML-stub'ом и ревью. Не пересекается с pin_media_addition_sessions
-- (та для добавления медиа в СУЩЕСТВУЮЩИЙ пин).
CREATE TABLE pin_creation_sessions (
 session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
 initiator_user_id UUID NOT NULL,
 draft_snapshot JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 closed_at TIMESTAMPTZ,
 close_reason TEXT
);
-- Одна активная сессия создания пина на трип (защита от race + параллельного over-upload).
CREATE UNIQUE INDEX idx_pin_creation_sessions_active_per_trip
 ON pin_creation_sessions(trip_id) WHERE closed_at IS NULL;

-- Связь media с pin-creation-сессией: до finalize media лежит с pin_id=NULL и
-- pin_creation_session_id; на finalize pin_id заполняется через UpdatePinIDByIDs,
-- pin_creation_session_id остаётся (не мешает другим запросам, но позволяет
-- собирать историю при необходимости). На cancel media удаляется по этому ключу.
ALTER TABLE media ADD COLUMN pin_creation_session_id UUID
 REFERENCES pin_creation_sessions(session_id) ON DELETE SET NULL;
CREATE INDEX idx_media_pin_creation_session ON media(pin_creation_session_id)
 WHERE pin_creation_session_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_media_pin_creation_session;
ALTER TABLE media DROP COLUMN IF EXISTS pin_creation_session_id;
DROP INDEX IF EXISTS idx_pin_creation_sessions_active_per_trip;
DROP TABLE IF EXISTS pin_creation_sessions;
