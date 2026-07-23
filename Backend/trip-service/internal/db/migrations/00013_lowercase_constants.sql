-- +goose Up
-- +goose StatementBegin
UPDATE trips         SET privacy_level = LOWER(privacy_level);
UPDATE pins          SET privacy_level = LOWER(privacy_level);
UPDATE media         SET privacy_level = LOWER(privacy_level);
UPDATE pin_privacy   SET privacy_level = LOWER(privacy_level);
UPDATE trip_privacy  SET privacy_level = LOWER(privacy_level);
UPDATE media_privacy SET privacy_level = LOWER(privacy_level);
UPDATE social        SET reaction      = LOWER(reaction);
UPDATE geo_registry  SET type          = LOWER(type);

UPDATE trips SET status = 'DRAFT' WHERE status IN ('Created', 'Draft');

UPDATE trips SET category = CASE category
    WHEN 'Отпуск'         THEN 'vacation'
    WHEN 'Командировка'   THEN 'business'
    WHEN 'Выходные'       THEN 'holidays'
    WHEN 'Активный отдых' THEN 'active'
    WHEN 'Образование'    THEN 'education'
    WHEN 'Другое'         THEN 'custom'
    ELSE category
END;

UPDATE trips SET season = CASE season
    WHEN 'Зима'  THEN 'winter'
    WHEN 'Весна' THEN 'spring'
    WHEN 'Лето'  THEN 'summer'
    WHEN 'Осень' THEN 'autumn'
    ELSE season
END;

UPDATE pins SET category = CASE category
    WHEN 'Достопримечательность' THEN 'sight'
    WHEN 'Природа'               THEN 'nature'
    WHEN 'Отдых'                 THEN 'leisure'
    WHEN 'Жилье'                 THEN 'housing'
    WHEN 'Еда и напитки'         THEN 'food'
    WHEN 'Шопинг'                THEN 'shopping'
    WHEN 'Транспорт'             THEN 'transport'
    WHEN 'Развлечение'           THEN 'entertainment'
    WHEN 'Мероприятие'           THEN 'event'
    WHEN 'Спорт'                 THEN 'sport'
    WHEN 'Рабочее место'         THEN 'work'
    WHEN 'Другое'                THEN 'custom'
    ELSE category
END;

ALTER TABLE trips ALTER COLUMN status        SET DEFAULT 'DRAFT';
ALTER TABLE trips ALTER COLUMN privacy_level SET DEFAULT 'private';
ALTER TABLE pins  ALTER COLUMN privacy_level SET DEFAULT 'private';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE trips ALTER COLUMN status        SET DEFAULT 'Created';
ALTER TABLE trips ALTER COLUMN privacy_level SET DEFAULT 'Private';
ALTER TABLE pins  ALTER COLUMN privacy_level SET DEFAULT 'Private';
-- +goose StatementEnd
