-- +goose Up
CREATE TABLE desired_places (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    image_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_desired_places_user_id_created_at
    ON desired_places(user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS desired_places;
