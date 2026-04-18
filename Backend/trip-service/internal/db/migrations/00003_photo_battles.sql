-- +goose Up
-- PINZ-132: фотобатлы (ТЗ 8.1). Таблица сессий батла: сервер фиксирует 8 выбранных медиа,
-- клиент проводит турнир локально и присылает итогового победителя; finished_at защищает от повторного инкремента.
CREATE TABLE media_battles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  media_ids JSONB NOT NULL,
  winner_media_id UUID REFERENCES media(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at TIMESTAMPTZ
);
CREATE INDEX media_battles_trip_id_idx ON media_battles(trip_id);
CREATE INDEX media_battles_user_id_idx ON media_battles(user_id);

-- +goose Down
DROP TABLE IF EXISTS media_battles;
