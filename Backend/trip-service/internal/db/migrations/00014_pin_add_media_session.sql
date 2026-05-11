-- +goose Up
ALTER TABLE pins ADD COLUMN add_media_session_id UUID NULL
    REFERENCES add_media_sessions(session_id) ON DELETE SET NULL;

CREATE INDEX idx_pins_add_media_session_id
    ON pins(add_media_session_id) WHERE add_media_session_id IS NOT NULL;

ALTER TABLE add_media_sessions
    ADD COLUMN pending_existing_attachments JSONB NOT NULL DEFAULT '[]'::jsonb;

-- +goose Down
ALTER TABLE add_media_sessions DROP COLUMN IF EXISTS pending_existing_attachments;
DROP INDEX IF EXISTS idx_pins_add_media_session_id;
ALTER TABLE pins DROP COLUMN IF EXISTS add_media_session_id;
