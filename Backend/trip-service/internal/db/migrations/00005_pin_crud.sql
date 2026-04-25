-- +goose Up
-- ТЗ 4.5.2: soft-delete пина для текущего юзера через pin_hidden_by_user.
-- PK по (pin_id, user_id) — для фильтра ListByTripID/SearchByUserID нужен индекс
-- по user_id (LEFT JOIN с фильтром будет частым).
CREATE INDEX IF NOT EXISTS pin_hidden_by_user_user_idx ON pin_hidden_by_user(user_id);

-- ТЗ 4.2.2 + 4.12-4.14: sessioned add-media-в-пин с этапом ML-stub и ревью.
-- Отдельная таблица — не пересекается с add_media_sessions (UNIQUE per-trip,
-- коллаборативный режим, initiator/takeover). Pin-сессия per-pin, без
-- collaboration. Один user ведёт сессию от start до finalize.
CREATE TABLE pin_media_addition_sessions (
 session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
 pin_id UUID NOT NULL REFERENCES pins(id) ON DELETE CASCADE,
 initiator_user_id UUID NOT NULL,
 draft_snapshot JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 closed_at TIMESTAMPTZ,
 close_reason TEXT
);
CREATE UNIQUE INDEX idx_pin_media_addition_sessions_active_per_pin
 ON pin_media_addition_sessions(pin_id) WHERE closed_at IS NULL;

-- Связь media с pin-add-сессией: до finalize media лежит с pin_id=NULL и
-- pin_addition_session_id — для orphan-cleanup при cancel и фильтрации в
-- Process/Review/GetTrip (чтобы draft media не попадали в обычные ответы).
ALTER TABLE media ADD COLUMN pin_addition_session_id UUID
 REFERENCES pin_media_addition_sessions(session_id) ON DELETE SET NULL;
CREATE INDEX idx_media_pin_addition_session ON media(pin_addition_session_id)
 WHERE pin_addition_session_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_media_pin_addition_session;
ALTER TABLE media DROP COLUMN IF EXISTS pin_addition_session_id;
DROP INDEX IF EXISTS idx_pin_media_addition_sessions_active_per_pin;
DROP TABLE IF EXISTS pin_media_addition_sessions;
DROP INDEX IF EXISTS pin_hidden_by_user_user_idx;
