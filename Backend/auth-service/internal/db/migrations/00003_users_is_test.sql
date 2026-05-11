-- +goose Up
ALTER TABLE users ADD COLUMN is_test BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX users_is_test_idx ON users (is_test) WHERE is_test = true;

-- +goose Down
DROP INDEX IF EXISTS users_is_test_idx;
ALTER TABLE users DROP COLUMN IF EXISTS is_test;
