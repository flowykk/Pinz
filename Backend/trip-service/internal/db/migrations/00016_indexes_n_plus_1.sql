-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_trip_id ON media(trip_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_pin_id ON media(pin_id) WHERE pin_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pins_trip_id ON pins(trip_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trip_participants_user_id ON trip_participants(user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trips_owner_user_id ON trips(owner_user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trips_updated_at_active
  ON trips(updated_at DESC)
  WHERE is_soft_deleted = false AND is_generated = false;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_media_trip_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_media_pin_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_pins_trip_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_trip_participants_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_trips_owner_user_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_trips_updated_at_active;
