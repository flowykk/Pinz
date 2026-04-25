-- +goose Up
-- +goose StatementBegin
-- device_tokens: APNS-токены устройств, зарегистрированные пользователем.
-- Один и тот же apns_token уникален — при повторной регистрации переносим его
-- на нового user_id (смена аккаунта на устройстве).
CREATE TABLE IF NOT EXISTS device_tokens (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id UUID NOT NULL,
 apns_token TEXT NOT NULL UNIQUE,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS device_tokens_user_id_idx
 ON device_tokens (user_id);

-- notification_log: журнал отправленных пушей для идемпотентности.
-- event_id (Redis msg id или уникальный scheduler ключ) + apns_token
-- уникальны — избегаем повторной доставки при рестартах/ретраях.
CREATE TABLE IF NOT EXISTS notification_log (
 event_id TEXT NOT NULL,
 apns_token TEXT NOT NULL,
 sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (event_id, apns_token)
);

CREATE INDEX IF NOT EXISTS notification_log_sent_at_idx
 ON notification_log (sent_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notification_log;
DROP TABLE IF EXISTS device_tokens;
-- +goose StatementEnd
