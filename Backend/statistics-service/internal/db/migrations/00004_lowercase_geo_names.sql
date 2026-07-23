-- +goose Up
-- +goose StatementBegin

-- Перед UPDATE name=LOWER(name) сливаем дубликаты, иначе уникальные индексы
-- geo_registry_country_uniq / geo_registry_city_uniq упадут на конфликт.
-- Канонической оставляем строку с минимальным id; ссылки в trip_locations
-- переводим на каноническую, дубль удаляем. Дети дублей-стран
-- (parent_id) переключаем на канонический parent.

-- 1. Дубли стран (parent_id IS NULL).
WITH dup_country AS (
 SELECT id,
        MIN(id) OVER (PARTITION BY LOWER(name), type) AS canonical_id
 FROM geo_registry
 WHERE parent_id IS NULL
), redirect AS (
 SELECT id, canonical_id FROM dup_country WHERE id <> canonical_id
)
UPDATE geo_registry g SET parent_id = r.canonical_id
FROM redirect r WHERE g.parent_id = r.id;

WITH dup_country AS (
 SELECT id,
        MIN(id) OVER (PARTITION BY LOWER(name), type) AS canonical_id
 FROM geo_registry
 WHERE parent_id IS NULL
), redirect AS (
 SELECT id, canonical_id FROM dup_country WHERE id <> canonical_id
)
UPDATE trip_locations t SET location_id = r.canonical_id
FROM redirect r WHERE t.location_id = r.id
AND NOT EXISTS (
 SELECT 1 FROM trip_locations t2
 WHERE t2.trip_id = t.trip_id AND t2.location_id = r.canonical_id
);

DELETE FROM trip_locations t
USING (
 SELECT id, MIN(id) OVER (PARTITION BY LOWER(name), type) AS canonical_id
 FROM geo_registry WHERE parent_id IS NULL
) d
WHERE t.location_id = d.id AND d.id <> d.canonical_id;

DELETE FROM geo_registry WHERE id IN (
 SELECT id FROM (
  SELECT id, MIN(id) OVER (PARTITION BY LOWER(name), type) AS canonical_id
  FROM geo_registry WHERE parent_id IS NULL
 ) d WHERE id <> canonical_id
);

-- 2. Дубли городов (parent_id IS NOT NULL).
WITH dup_city AS (
 SELECT id,
        MIN(id) OVER (PARTITION BY LOWER(name), type, parent_id) AS canonical_id
 FROM geo_registry
 WHERE parent_id IS NOT NULL
), redirect AS (
 SELECT id, canonical_id FROM dup_city WHERE id <> canonical_id
)
UPDATE trip_locations t SET location_id = r.canonical_id
FROM redirect r WHERE t.location_id = r.id
AND NOT EXISTS (
 SELECT 1 FROM trip_locations t2
 WHERE t2.trip_id = t.trip_id AND t2.location_id = r.canonical_id
);

DELETE FROM trip_locations t
USING (
 SELECT id, MIN(id) OVER (PARTITION BY LOWER(name), type, parent_id) AS canonical_id
 FROM geo_registry WHERE parent_id IS NOT NULL
) d
WHERE t.location_id = d.id AND d.id <> d.canonical_id;

DELETE FROM geo_registry WHERE id IN (
 SELECT id FROM (
  SELECT id, MIN(id) OVER (PARTITION BY LOWER(name), type, parent_id) AS canonical_id
  FROM geo_registry WHERE parent_id IS NOT NULL
 ) d WHERE id <> canonical_id
);

-- 3. Привести оставшиеся имена к lower-case.
UPDATE geo_registry SET name = LOWER(name) WHERE name <> LOWER(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
