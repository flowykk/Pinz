-- +goose Up
-- async ML для add-media-в-пин: стадии UPLOADING → PROCESSING → READY_FOR_REVIEW.
ALTER TABLE pin_media_addition_sessions
 ADD COLUMN processing_status TEXT NOT NULL DEFAULT 'UPLOADING';

ALTER TABLE pin_media_addition_sessions
 ADD CONSTRAINT pin_media_addition_sessions_processing_status_check
 CHECK (processing_status IN ('UPLOADING', 'PROCESSING', 'READY_FOR_REVIEW'));

-- +goose Down
ALTER TABLE pin_media_addition_sessions DROP CONSTRAINT IF EXISTS pin_media_addition_sessions_processing_status_check;
ALTER TABLE pin_media_addition_sessions DROP COLUMN IF EXISTS processing_status;
