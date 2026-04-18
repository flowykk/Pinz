-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_stats (
    user_id UUID PRIMARY KEY,
    total_likes      INT NOT NULL DEFAULT 0,
    total_dislikes   INT NOT NULL DEFAULT 0,
    battles_finished INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS geo_registry (
    id        INT PRIMARY KEY,
    parent_id INT,
    name      TEXT NOT NULL,
    type      TEXT NOT NULL
);

-- trip_locations — зеркало локаций трипа (независимо от участников).
-- GetVisitedLocations агрегирует по списку trip_ids, который API Gateway
-- получает из trip-service для текущего пользователя.
CREATE TABLE IF NOT EXISTS trip_locations (
    trip_id     UUID NOT NULL,
    location_id INT  NOT NULL REFERENCES geo_registry(id),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (trip_id, location_id)
);

CREATE INDEX IF NOT EXISTS trip_locations_location_idx
    ON trip_locations (location_id);

CREATE TABLE IF NOT EXISTS stats_event_log (
    event_id     TEXT PRIMARY KEY,
    event_type   TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS stats_event_log_processed_at_idx
    ON stats_event_log (processed_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS stats_event_log;
DROP TABLE IF EXISTS trip_locations;
DROP TABLE IF EXISTS geo_registry;
DROP TABLE IF EXISTS user_stats;
-- +goose StatementEnd
