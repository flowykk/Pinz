-- +goose Up
-- +goose StatementBegin

-- geo_registry становится read-only репликой statistics-service:
-- id больше не выдаётся локально (приходит из master), уникальные индексы
-- по name больше не нужны (mirror идёт по id с ON CONFLICT (id) DO UPDATE).
DROP INDEX IF EXISTS geo_registry_city_uniq;
DROP INDEX IF EXISTS geo_registry_country_uniq;
ALTER TABLE geo_registry ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE IF EXISTS geo_registry_id_seq;

-- Лог обработанных событий из pinz:trip:geo_events для идемпотентности consumer'а.
CREATE TABLE IF NOT EXISTS geo_event_log (
 event_id TEXT PRIMARY KEY,
 event_type TEXT NOT NULL,
 processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS geo_event_log_processed_at_idx
 ON geo_event_log (processed_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS geo_event_log;

-- Восстановление writable-режима (для отката миграции).
CREATE SEQUENCE IF NOT EXISTS geo_registry_id_seq;
SELECT setval('geo_registry_id_seq', GREATEST(COALESCE((SELECT MAX(id) FROM geo_registry), 0), 1));
ALTER TABLE geo_registry ALTER COLUMN id SET DEFAULT nextval('geo_registry_id_seq');
CREATE UNIQUE INDEX IF NOT EXISTS geo_registry_country_uniq ON geo_registry (name, type) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS geo_registry_city_uniq ON geo_registry (name, type, parent_id) WHERE parent_id IS NOT NULL;
-- +goose StatementEnd
