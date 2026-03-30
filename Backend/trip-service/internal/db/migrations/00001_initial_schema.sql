-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE trips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id UUID NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  category TEXT NOT NULL,
  season TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'Created',
  privacy_level TEXT NOT NULL DEFAULT 'Private',
  start_date TIMESTAMPTZ,
  end_date TIMESTAMPTZ,
  likes_count INT NOT NULL DEFAULT 0,
  dislikes_count INT NOT NULL DEFAULT 0,
  cover_url TEXT,
  is_published BOOLEAN NOT NULL DEFAULT false,
  is_generated BOOLEAN NOT NULL DEFAULT false,
  is_soft_deleted BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE trip_participants (
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  is_admin BOOLEAN NOT NULL DEFAULT false,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (trip_id, user_id)
);

CREATE TABLE invitation_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE trip_settings (
  user_id UUID NOT NULL,
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  notifications_enabled BOOLEAN NOT NULL DEFAULT true,
  PRIMARY KEY (user_id, trip_id)
);

CREATE TABLE trip_privacy (
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  privacy_level TEXT NOT NULL,
  PRIMARY KEY (trip_id, user_id)
);

CREATE TABLE pins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  location GEOMETRY(Point, 4326),
  category TEXT NOT NULL,
  privacy_level TEXT NOT NULL DEFAULT 'Private',
  media_count INT NOT NULL DEFAULT 0,
  start_time TIMESTAMPTZ,
  end_time TIMESTAMPTZ,
  is_published_in_feed BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE media (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  pin_id UUID REFERENCES pins(id) ON DELETE SET NULL,
  s3_key TEXT NOT NULL,
  media_type TEXT NOT NULL,
  location GEOMETRY(Point, 4326),
  captured_at TIMESTAMPTZ,
  battle_rating INT NOT NULL DEFAULT 0,
  privacy_level TEXT NOT NULL DEFAULT 'Private',
  similar_group_id UUID,
  content_hash TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  pin_id UUID REFERENCES pins(id) ON DELETE CASCADE,
  tag TEXT NOT NULL,
  UNIQUE(trip_id, pin_id, tag)
);

CREATE TABLE pin_privacy (
  pin_id UUID NOT NULL REFERENCES pins(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  privacy_level TEXT NOT NULL,
  PRIMARY KEY (pin_id, user_id)
);

CREATE TABLE media_privacy (
  media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  privacy_level TEXT NOT NULL,
  PRIMARY KEY (media_id, user_id)
);

CREATE TABLE social (
  user_id UUID NOT NULL,
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  reaction TEXT NOT NULL,
  PRIMARY KEY (user_id, trip_id)
);

CREATE TABLE favourite (
  user_id UUID NOT NULL,
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, trip_id)
);

CREATE TABLE geo_registry (
  id INT PRIMARY KEY,
  parent_id INT REFERENCES geo_registry(id),
  name TEXT NOT NULL,
  type TEXT NOT NULL
);

CREATE TABLE trip_locations (
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  location_id INT NOT NULL REFERENCES geo_registry(id),
  PRIMARY KEY (trip_id, location_id)
);

CREATE TABLE add_media_sessions (
  session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  existing_media_ids JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE pin_hidden_by_user (
  pin_id UUID NOT NULL REFERENCES pins(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  PRIMARY KEY (pin_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS pin_hidden_by_user;
DROP TABLE IF EXISTS add_media_sessions;
DROP TABLE IF EXISTS trip_locations;
DROP TABLE IF EXISTS geo_registry;
DROP TABLE IF EXISTS favourite;
DROP TABLE IF EXISTS social;
DROP TABLE IF EXISTS media_privacy;
DROP TABLE IF EXISTS pin_privacy;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS media;
DROP TABLE IF EXISTS pins;
DROP TABLE IF EXISTS trip_privacy;
DROP TABLE IF EXISTS trip_settings;
DROP TABLE IF EXISTS invitation_links;
DROP TABLE IF EXISTS trip_participants;
DROP TABLE IF EXISTS trips;
DROP EXTENSION IF EXISTS postgis CASCADE;
