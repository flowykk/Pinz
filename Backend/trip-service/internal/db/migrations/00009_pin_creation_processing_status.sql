-- +goose Up
-- async ML для pin-creation: стадии UPLOADING → PROCESSING → READY_FOR_REVIEW.
ALTER TABLE pin_creation_sessions
 ADD COLUMN processing_status TEXT NOT NULL DEFAULT 'UPLOADING';

ALTER TABLE pin_creation_sessions
 ADD CONSTRAINT pin_creation_sessions_processing_status_check
 CHECK (processing_status IN ('UPLOADING', 'PROCESSING', 'READY_FOR_REVIEW'));

-- +goose Down
ALTER TABLE pin_creation_sessions DROP CONSTRAINT IF EXISTS pin_creation_sessions_processing_status_check;
ALTER TABLE pin_creation_sessions DROP COLUMN IF EXISTS processing_status;
