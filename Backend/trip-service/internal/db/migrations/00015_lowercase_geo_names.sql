-- +goose Up
-- +goose StatementBegin

-- geo_registry здесь read-only реплика, уникальных индексов по name нет
-- (сняты в 00008), поэтому достаточно UPDATE без дедупликации:
-- следующий PIN_LOCATIONS_RESOLVED доперезапишет строки канонически.
UPDATE geo_registry SET name = LOWER(name) WHERE name <> LOWER(name);
UPDATE pins SET location_name = LOWER(location_name)
 WHERE location_name IS NOT NULL AND location_name <> LOWER(location_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
