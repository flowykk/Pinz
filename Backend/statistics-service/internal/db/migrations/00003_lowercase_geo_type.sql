-- +goose Up
-- +goose StatementBegin
UPDATE geo_registry SET type = LOWER(type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
