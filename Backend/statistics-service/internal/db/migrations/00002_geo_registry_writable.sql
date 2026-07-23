-- +goose Up
-- +goose StatementBegin

-- Statistics-service становится владельцем GEO_REGISTRY:
-- id выдаётся локально через sequence, уникальные индексы по name защищают
-- идемпотентный upsert "EnsureByName" в обработчике PIN_LOCATIONS_REQUESTED.
CREATE SEQUENCE IF NOT EXISTS geo_registry_id_seq;
SELECT setval('geo_registry_id_seq', GREATEST(COALESCE((SELECT MAX(id) FROM geo_registry), 0), 1));
ALTER TABLE geo_registry ALTER COLUMN id SET DEFAULT nextval('geo_registry_id_seq');

CREATE UNIQUE INDEX IF NOT EXISTS geo_registry_country_uniq
 ON geo_registry (name, type) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS geo_registry_city_uniq
 ON geo_registry (name, type, parent_id) WHERE parent_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS geo_registry_city_uniq;
DROP INDEX IF EXISTS geo_registry_country_uniq;
ALTER TABLE geo_registry ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE IF EXISTS geo_registry_id_seq;
-- +goose StatementEnd
