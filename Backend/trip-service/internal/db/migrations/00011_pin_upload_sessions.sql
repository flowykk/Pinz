-- +goose Up
-- pin_upload_sessions: target_pin_id NULL = создание пина, заполнен = добавление медиа в пин.
CREATE TABLE pin_upload_sessions (
 session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
 target_pin_id UUID NULL REFERENCES pins(id) ON DELETE CASCADE,
 initiator_user_id UUID NOT NULL,
 draft_snapshot JSONB,
 processing_status TEXT NOT NULL DEFAULT 'UPLOADING'
   CHECK (processing_status IN ('UPLOADING', 'PROCESSING', 'READY_FOR_REVIEW')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 closed_at TIMESTAMPTZ,
 close_reason TEXT
);

CREATE UNIQUE INDEX idx_pin_upload_sessions_active_creation
 ON pin_upload_sessions(trip_id) WHERE target_pin_id IS NULL AND closed_at IS NULL;

CREATE UNIQUE INDEX idx_pin_upload_sessions_active_addition
 ON pin_upload_sessions(target_pin_id) WHERE target_pin_id IS NOT NULL AND closed_at IS NULL;

CREATE INDEX idx_pin_upload_sessions_target_pin
 ON pin_upload_sessions(target_pin_id) WHERE target_pin_id IS NOT NULL;

ALTER TABLE media ADD COLUMN upload_session_id UUID
 REFERENCES pin_upload_sessions(session_id) ON DELETE SET NULL;
CREATE INDEX idx_media_upload_session ON media(upload_session_id)
 WHERE upload_session_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_media_upload_session;
ALTER TABLE media DROP COLUMN IF EXISTS upload_session_id;
DROP INDEX IF EXISTS idx_pin_upload_sessions_target_pin;
DROP INDEX IF EXISTS idx_pin_upload_sessions_active_addition;
DROP INDEX IF EXISTS idx_pin_upload_sessions_active_creation;
DROP TABLE IF EXISTS pin_upload_sessions;
