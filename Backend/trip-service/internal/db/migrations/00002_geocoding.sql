-- +goose Up
ALTER TABLE pins ADD COLUMN location_name TEXT NOT NULL DEFAULT '';

CREATE SEQUENCE IF NOT EXISTS geo_registry_id_seq;
SELECT setval('geo_registry_id_seq', GREATEST(COALESCE((SELECT MAX(id) FROM geo_registry), 0), 1));
ALTER TABLE geo_registry ALTER COLUMN id SET DEFAULT nextval('geo_registry_id_seq');

-- Partial unique indexes: NULL parent_id (страны) и NOT NULL parent_id (города) обрабатываются отдельно,
-- т.к. PostgreSQL считает NULL != NULL в обычном UNIQUE constraint.
CREATE UNIQUE INDEX IF NOT EXISTS geo_registry_country_uniq ON geo_registry (name, type) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS geo_registry_city_uniq ON geo_registry (name, type, parent_id) WHERE parent_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS geo_registry_city_uniq;
DROP INDEX IF EXISTS geo_registry_country_uniq;
ALTER TABLE geo_registry ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE IF EXISTS geo_registry_id_seq;
ALTER TABLE pins DROP COLUMN IF EXISTS location_name;
