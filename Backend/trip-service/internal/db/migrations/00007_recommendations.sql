-- +goose Up
-- ТЗ 9: рекомендательная система. Индексы под выборку топ-трипов региона
-- и пинов для последующей кластеризации через ST_ClusterDBSCAN.
CREATE INDEX IF NOT EXISTS pins_location_gist ON pins USING GIST (location);
CREATE INDEX IF NOT EXISTS trip_locations_location_id_idx ON trip_locations(location_id);
CREATE INDEX IF NOT EXISTS trips_published_score_idx
 ON trips ((likes_count - dislikes_count) DESC)
 WHERE is_published = true AND is_soft_deleted = false;

-- +goose Down
DROP INDEX IF EXISTS trips_published_score_idx;
DROP INDEX IF EXISTS trip_locations_location_id_idx;
DROP INDEX IF EXISTS pins_location_gist;
