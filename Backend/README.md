[# Pinz Backend

Микросервисный бэкенд приложения Pinz: авторизация, passkey, JWT, observability.

## Архитектура

```
                     ┌──────────────────────────────────────┐
                     │           Kubernetes (Minikube)       │
                     │                                       │
  Client ──► Istio ──► api-gateway ──gRPC──► auth-service   │
                     │                                       │
                     │  otel-collector  tempo  grafana       │
                     │  prometheus      loki                 │
                     └──────────────────────────────────────┘
                                    │
                     ┌──────────────┴──────────────┐
                     │     Docker Compose (host)    │
                     │     PostgreSQL    Redis       │
                     └─────────────────────────────┘
```

| Компонент | Роль |
|---|---|
| **api-gateway-service** | REST → gRPC прокси, HTTP-трейсинг, WebSocket |
| **auth-service** | Бизнес-логика авторизации, passkey, JWT |
| **trip-service** | Путешествия, пины, медиа, участники, лента, async-флоу |
| **statistics-service** | Счётчики пользователя (трипы, пины, медиа, батлы), посещённые локации (consumer Redis Streams `pinz:stats:events`) |
| **notification-service** | Push через APNS + email через SMTP (consumer `pinz:trip:events`, `pinz:auth:email:tasks`); scheduler для годовщины трипа и «1 месяц после end_date» |
| **otel-collector** | Приём и маршрутизация телеметрии (OTLP) |
| **tempo** | Хранение распределённых трейсов |
| **prometheus** | Метрики (RED + бизнес + Go runtime) |
| **loki** | Структурированные логи |
| **promtail** | Логи контейнеров с нод → Loki |
| **grafana** | Дашборды, корреляция трейс↔лог↔метрика |

## Стек

Go 1.25 · chi · gRPC · Protobuf · PostgreSQL · Redis · JWT · OpenTelemetry · Istio · Helm · Helmfile

## ML интеграция

Trip-service общается с внешним ML-сервисом только через Redis Streams и presigned S3 GET URLs — ML не имеет прямого доступа к Postgres и S3-ключам. Контракт стримов (задачи и результаты), правила для всех трёх сценариев (Trip Creation, Add Media, Pin Upload), формат payload, SLA задокументированы отдельно.

Стримы:

| Стрим | Направление | Сценарии |
|---|---|---|
| `pinz:trip:ml:tasks` | backend → ML | Trip Creation, Add Media |
| `pinz:trip:ml:results` | ML → backend | Trip Creation, Add Media |
| `pinz:trip:pin_upload:ml:tasks` | backend → ML | Pin Upload |
| `pinz:trip:pin_upload:ml:results` | ML → backend | Pin Upload |

Билдеры payload'а — `internal/services/ml_payload.go` (`BuildTripMLPayload`, `BuildPinUploadMLPayload`). Worker `internal/worker/pin_upload_ml_consumer.go` потребляет результаты pin-upload и переводит сессию в `READY_FOR_REVIEW`. Активация интеграции (снятие stub'ов `finalizeProcessingStub` и переключение pin-upload-consumer на ML-step) — отдельная задача.

## Схема БД

Миграции [goose](https://github.com/pressly/goose) — SQL в `auth-service/internal/db/migrations/` и `trip-service/internal/db/migrations/`, применяются при старте. Запросы к БД — [sqlc](https://docs.sqlc.dev/): `queries/` и сгенерированный код в `internal/db/sqlcdb/` (схема из тех же миграций). Из каталога `Backend`: `make sqlc`, в CI — `make sqlc-check`.

## Эндпоинты (REST)

### Auth (auth-service через API Gateway)

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/auth/email` | Ввод почты. Ответ: `{"is_registered": bool, "registration_key": "..."}` |
| POST | `/api/v1/auth/verify-email` | Проверка кода из письма |
| POST | `/api/v1/auth/finish-register` | Установка пароля и username |
| POST | `/api/v1/auth/login` | Вход по email + пароль |
| POST | `/api/v1/auth/refresh` | Обновление access-токена |
| POST | `/api/v1/auth/logout` | Выход |

### Trip creation flow (trip-service через API Gateway)

| Шаг | Метод | Путь | Описание |
|---|---|---|---|
| 1 | POST | `/api/v1/trips/creation/start` | Создание путешествия + выдача presigned PUT URL для загрузки в Object Storage (S3‑совместимый API) |
| 2 | POST | `/api/v1/trips/creation/{trip_id}/media/process-grouping` | Передача метаданных медиа, первичная группировка по гео/времени, статус `DRAFT_GROUPING_REVIEW` |
| 3 | POST | `/api/v1/trips/creation/{trip_id}/apply-groups-and-process` | Применение ручных групп, создание пинов, статус `PROCESSING`, задача в Redis Streams `pinz:trip:ml:tasks` |
| 4 | GET | `/api/v1/trips/creation/{trip_id}/review` | Результат фоновой обработки: пины, теги, `issues`, массив `similar` (похожие медиа) |
| 5 | POST | `/api/v1/trips/creation/{trip_id}/finalize` | Финальное ревью, ручные правки, удаление медиа, агрегация трипа, статус `READY` |

**Object Storage (trip-service):** используется Yandex Cloud Object Storage (или любой S3‑совместимый endpoint). Переменные окружения: `S3_ENDPOINT` (по умолчанию `https://storage.yandexcloud.net`), `S3_REGION` (`ru-central1`), `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, опционально `S3_PRESIGN_TTL` (например `15m`). Если `S3_BUCKET` не задан, presigned URL не выдаются (`url` в ответе пустой). В консоли YC: сервисный аккаунт с ролью `storage.editor`, статический ключ доступа, бакет; для прямой загрузки с клиента настройте CORS на бакете (методы `GET`, `PUT`, `HEAD`).

Real‑time уведомление о завершении шага 3–4 идёт через WebSocket:

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/trips/creation/{trip_id}/review/ws` | WebSocket для стадии ревью флоу создания путешествия. JWT в `Authorization: Bearer <token>` handshake‑запроса. До апгрейда gateway вызывает `TripService.GetTrip` и отвечает `403` / `404`, если пользователь не участник или трипа нет. Сервер подписывается на `pinz:trip:{trip_id}:events` (изолированный канал на трип — параллельное создание нескольких путешествий не смешивается). Формат сообщения: `{"event":"TRIP_PROCESSING_COMPLETED","payload":{"trip_id":"...","status":"DRAFT_FINAL_REVIEW"}}`. Heartbeat: сервер шлёт `ping` каждые 30 с, клиент обязан отвечать `pong` (URLSession/браузеры делают автоматически). |
| GET | `/api/v1/trips/{trip_id}/media/add/review/ws` | WebSocket для стадии ревью флоу добавления медиа в существующее путешествие (ТЗ 5.3). Контракт, авторизация и heartbeat — те же, что у `creation/{id}/review/ws`; подписка на тот же `pinz:trip:{trip_id}:events`. |
| GET | `/api/v1/trips/{trip_id}/pin-uploads/{sid}/ws` | WebSocket для async-обработки унифицированной pin-upload сессии (creation либо addition в существующий пин — определяется `target_pin_id`). Событие `PIN_UPLOAD_PROCESSING_COMPLETED`. |

Воркер `trip-service` публикует WS-события через `PublishTripEventWS` в `pinz:trip:{trip_id}:events`. Per-resource WS-endpoint'ы фильтруют события по соответствующему `payload.session_id`/`payload.pin_id`.

### Trip core / feed (trip-service через API Gateway)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/trips` | Список своих путешествий (учитывая участие в трипах) |
| POST | `/api/v1/trips` | Создание путешествия (тот же gRPC, что и `trips/creation/start`; основной путь — `trips/creation/start`) |
| GET | `/api/v1/trips/{id}` | Детали путешествия (только участники или те, у кого в избранном при soft‑delete) |
| PATCH | `/api/v1/trips/{id}` | Редактирование параметров путешествия (название, описание, категория, сезон, даты, приватность). Обложка — через `/cover/*` ниже |
| DELETE | `/api/v1/trips/{id}` | Удаление путешествия с учётом избранного (полное или soft‑delete; см. PINZ‑98) |
| POST | `/api/v1/trips/{id}/cover/upload` | Step 1 обложки: presigned PUT URL для загрузки в S3 (аналог user avatar) |
| POST | `/api/v1/trips/{id}/cover/confirm` | Step 2 обложки: подтверждение по `s3_key`, обновление `cover_url`, best-effort удаление старого объекта |
| DELETE | `/api/v1/trips/{id}/cover` | Удаление обложки (best-effort из S3 + очистка `cover_url`) |
| POST | `/api/v1/trips/{id}/invite` | Генерация инвайт‑ссылки (участники) |
| POST | `/api/v1/trips/join` | Присоединение к путешествию по токену инвайта |
| POST | `/api/v1/trips/{id}/leave` | Выход из путешествия; при уходе единственного админа трип удаляется или назначается новый админ (по ТЗ) |
| DELETE | `/api/v1/trips/{id}/participants/{user_id}` | Удаление участника (только админ) |
| PATCH | `/api/v1/trips/{id}/settings` | Вкл/выкл уведомлений по путешествию (ТЗ 12.4.1, PINZ‑98) |
| POST | `/api/v1/trips/{id}/publish` | Публикация путешествия в общую ленту целиком или по выбранным пинам (PINZ‑105) |
| GET | `/api/v1/feed` | Общая лента опубликованных путешествий (фильтры: категория, сезон, локация; сортировка по дате/рейтингу) |
| POST | `/api/v1/trips/{id}/like` | Лайк путешествия |
| POST | `/api/v1/trips/{id}/dislike` | Дизлайк путешествия |
| POST | `/api/v1/trips/{id}/favourite` | Добавить путешествие в избранное |
| DELETE | `/api/v1/trips/{id}/favourite` | Удалить путешествие из избранного |

### Statistics (statistics-service через API Gateway)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/profile/stats` | Счётчики текущего пользователя: трипы, пины, медиа, завершённые батлы |
| GET | `/api/v1/profile/visited-locations` | Список посещённых стран/городов; опц. `?type=country\|city` |

### Notifications (notification-service через API Gateway, PINZ-134)

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/profile/device-tokens` | Регистрация APNS-токена устройства: `{"apns_token":"..."}`. Повторная регистрация переносит токен на нового пользователя. Возвращает `token_id`. |
| DELETE | `/api/v1/profile/device-tokens` | Удаление APNS-токена устройства (logout на iOS): `{"apns_token":"..."}`. |

`notification-service` отправляет push через APNS по событиям `pinz:trip:events` (PARTICIPANT_JOINED/LEFT/REMOVED, ADMIN_CHANGED, TRIP_READY, PIN_ADDED) и таймерно через scheduler (годовщина трипа, 1 месяц после окончания). SMTP-отправка писем с кодами верификации (`pinz:auth:email:tasks`, публикует auth-service) также перенесена в этот сервис.

### Photo battles (trip-service через API Gateway, PINZ-132, ТЗ 8)

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/trips/{id}/battles` | Старт фотобатла: сервер случайно выбирает 8 медиа трипа (исключая `Restricted`) и возвращает `battle_id` + массив из 8 `{media_id, url, media_type}`. `412/409 FailedPrecondition`, если доступных медиа < 8 (ТЗ 8.1.9). Пары 4→2→1 клиент формирует локально. |
| POST | `/api/v1/trips/{id}/battles/{battle_id}/result` | Финал батла: `{"winner_media_id":"..."}`. Валидирует принадлежность победителя к выборке, атомарно закрывает сессию (`finished_at = NOW()`) и увеличивает `battle_rating` победителя на 1 (ТЗ 8.1.8). Повторный вызов — `409 FailedPrecondition: battle already finished`. |
| GET | `/api/v1/trips/{id}/best-memories` | «Лучшие воспоминания» (story-mode, ТЗ 8.2): медиа трипа с `battle_rating > 0`, отсортированные по рейтингу DESC. Пустой массив, если победителей ещё нет (ТЗ 8.2.3 — решение скрыть режим принимает клиент). |
| GET | `/api/v1/pins/search?q=&limit=&offset=` | Поиск пинов (PINZ-135) по `name`/`description`/`tags` в трипах, где пользователь — участник. `q` обязателен (1..128 симв.), `limit` 1..100 (по умолчанию 20), `offset` ≥0. Возвращает массив `TripPin` с `trip_id`, координатами, тегами и медиа. Скрытые `pin_hidden_by_user` записи отфильтрованы (ТЗ 4.5.2). Поиск по ML-контенту (embeddings) — отдельная задача. |

Все три эндпоинта требуют JWT и участия в трипе (`PermissionDenied` → 403 для не-участников). Состояние батла хранится в таблице `media_battles` (`trip-service/internal/db/migrations/00003_photo_battles.sql`); `battle_rating` — колонка `media.battle_rating` и влияет на сортировку топ-медиа в ленте.

### Pin Create + RUD + Media (trip-service через API Gateway, ТЗ §4)

Полный CRUD пинов на READY-трипе: создание (sessioned флоу с ML-stub'ом и ревью), чтение/обновление/удаление, добавление/удаление медиа в существующий пин. Privacy (4.2.10) — отдельная ручка `PUT /api/v1/trips/{id}/pins/{pin_id}/privacy` (PINZ-154); поиск (4.4) — `/api/v1/pins/search` выше.

#### Создание пина (ТЗ 4.1, 4.6-4.11)

Унифицированный sessioned-флоу `/pin-uploads/`. `target_pin_id` в теле `start` определяет сценарий:

- `target_pin_id = null` → создание нового пина (UNIQUE per trip).
- `target_pin_id = "<pin_id>"` → добавление медиа в существующий пин (UNIQUE per pin).

ML-обработка асинхронная: `process` возвращает 202 + `processing_status: "PROCESSING"`, worker делает hash-дедуп, suggested-поля для creation, pin issues, переводит сессию в `READY_FOR_REVIEW` и публикует WS-событие `PIN_UPLOAD_PROCESSING_COMPLETED` в per-trip stream (фильтрация по `session_id` на стороне gateway).

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/trips/{id}/pin-uploads/start` | Старт сессии. Тело: `{target_pin_id?, files_to_upload}`. Возвращает `session_id` + presigned PUT URLs. `409`, если активная сессия для трипа/пина уже есть. `412 + WRONG_STATUS` если `trip.status != READY`. |
| POST | `/api/v1/trips/{id}/pin-uploads/{sid}/upload-urls` | Догрузка дополнительных presigned URLs. |
| POST | `/api/v1/trips/{id}/pin-uploads/{sid}/commit-upload` | После успешного PUT в S3: создаётся `media` с `pin_id=NULL` и `upload_session_id=$sid`. Лимиты трипа (≤500 media, ≤50 video). |
| POST | `/api/v1/trips/{id}/pin-uploads/{sid}/process` | Запуск async ML. CAS UPLOADING→PROCESSING + публикация задачи в `pinz:trip:pin_upload:tasks`. **HTTP 202**. |
| GET | `/api/v1/trips/{id}/pin-uploads/{sid}/review` | Snapshot сессии. Поля `draft`/`similar` заполнены только в `READY_FOR_REVIEW`. Для creation в `draft.suggested` — имя/категория/теги/координаты/start-end. Для addition `suggested = null`. |
| POST | `/api/v1/trips/{id}/pin-uploads/{sid}/finalize` | Только в `READY_FOR_REVIEW`. Для creation создаёт запись `pins` с правками поверх suggested, привязывает media, теги, geocoding, `PIN_ADDED`. Для addition — UpdatePinIDByIDs + IncMediaCount + пересчёт агрегатов + geocoding если у пина появились координаты впервые. Закрывает сессию. |
| POST | `/api/v1/trips/{id}/pin-uploads/{sid}/cancel` | Orphan-cleanup (`DeleteOrphanByUploadSession` + S3) + close. |

Состояние сессии — таблица `pin_upload_sessions` с двумя partial UNIQUE-индексами: `(trip_id) WHERE target_pin_id IS NULL AND closed_at IS NULL` и `(target_pin_id) WHERE target_pin_id IS NOT NULL AND closed_at IS NULL`. Связь media — колонка `media.upload_session_id` с `ON DELETE SET NULL`. Snapshot — JSONB. Cron `RunPinUploadCleanup` (interval 15 мин): закрывает заброшенные сессии (>72ч без активности, reason=abandoned) с orphan-cleanup в БД и S3, плюс физически удаляет finalized/cancelled-записи старше 30 дней — чтобы таблица `pin_upload_sessions` не разрасталась.

#### RUD пина и удаление одиночного медиа (ТЗ §4.2-4.5)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/trips/{id}/pins/{pin_id}` | Все поля пина: media + tags + privacy_level (ТЗ 4.3). Доступ — participant трипа или favourite-юзер. Если пин скрыт для caller'а через `pin_hidden_by_user` (soft-delete-for-self) → 404. |
| PATCH | `/api/v1/trips/{id}/pins/{pin_id}` | Изменение полей: name (≤100), description (≤5000), category (из ТЗ 2.2.4), latitude/longitude, start/end_time_unix, tags (replace-all при `tags_set=true`). Любой participant. Trip должен быть в READY. При смене координат — асинхронный reverse-geocoding через statistics-service. |
| DELETE | `/api/v1/trips/{id}/pins/{pin_id}` | Удаление пина. Если трип в избранном у других пользователей — soft-delete-for-self через `pin_hidden_by_user` (ТЗ 4.5.2). Иначе full delete с каскадом media (S3 cleanup), тегов и самого пина. Защита: запрет удаления при активной addition-сессии на этом пине. |
| DELETE | `/api/v1/trips/{id}/pins/{pin_id}/media/{media_id}` | Sessionless удаление одного медиа из пина с пересчётом агрегатов и S3 cleanup. Защита: пин не может остаться без медиа (ТЗ 2.2.9). |

Swagger UI: `http://pinz.example.com/swagger/index.html`

## Локальная разработка (Minikube + Istio)

### Требования

- Minikube, Docker, kubectl
- Helm, Helmfile
- istioctl

### Первоначальная настройка кластера (один раз)

```bash
minikube start --cpus 4 --memory 4096 --driver=docker
minikube addons enable metrics-server
istioctl install --set profile=demo -y
kubectl label namespace default istio-injection=enabled
```

### Запуск полного стека

```bash
make all-up
```

### Остановка

```bash
make all-down
```

### Адреса

| Ресурс | URL / команда |
|---|---|
| API Gateway | `http://localhost:8080` (после `make dev`) |
| Swagger UI | `http://localhost:8080/swagger/index.html` |
| **Grafana** | `make grafana` → http://localhost:3000 |
| Логи | `kubectl logs -f deployment/api-gateway` |
| Трейсы | Grafana → Explore → Tempo |
| Метрики | Grafana → Dashboards → Pinz — Service Overview |

### /etc/hosts (один раз)

```bash
sudo sed -i '' '/pinz.example.com/d' /etc/hosts
echo "127.0.0.1 pinz.example.com" | sudo tee -a /etc/hosts
```

### make dev

```bash
make dev    # api-gateway → http://localhost:8080 (Ctrl+C для остановки)
```

## Make команды

### Инфраструктура

```bash
make infra-up       # PostgreSQL + Redis (Docker Compose)
make infra-down     # Остановка инфраструктуры
```

### Observability

```bash
make obs-up         # Observability-стек в k8s
make obs-down       # Удалить из k8s
make obs-status     # Статус obs-подов
make grafana        # port-forward → http://localhost:3000
```

### Kubernetes

```bash
make k8s-build      # Сборка образов внутри Minikube
make k8s-istio      # Применить Istio-ресурсы (mTLS, Gateway, VirtualService)
make k8s-deploy     # Деплой через helmfile + rollout restart
```

### Кодогенерация и линтинг

```bash
make proto          # Генерация protobuf (Backend/proto/*.proto)
make swagger        # Генерация swagger (api-gateway-service)
make sqlc           # sqlc (auth-service, trip-service)
make sqlc-check     # sqlc + проверка артефактов (как в CI)
make lint           # Запустить golangci-lint для обоих сервисов
make lint-api       # Только api-gateway-service
make lint-auth      # Только auth-service
```

## Production развертывание (VPS)

### Полная настройка сервера

```bash
# На чистом сервере Ubuntu 22.04
wget https://raw.githubusercontent.com/flowykk/Pinz/main/Backend/setup-server.sh
chmod +x setup-server.sh
./setup-server.sh --repo-url https://github.com/flowykk/Pinz.git
```

Скрипт установит Docker, k3s, Helm, Helmfile, Istio, склонирует репозиторий и настроит инфраструктуру.

### Адреса (Production)

| Ресурс | URL |
|---|---|
| API Gateway | https://pinz.website |
| Swagger UI | https://pinz.website/swagger/index.html |
| Health check | `curl https://pinz.website/health` |
| **Grafana** | https://grafana.pinz.website (дашборды, трейсы, логи, метрики) |
| **Kiali** | https://kiali.pinz.website (service mesh graph, mTLS, конфиг Istio) |

### Деплой

```bash
cd /opt/pinz/Backend

# Ручной деплой
./deploy.sh

# Деплой с конкретным тегом
./deploy.sh --image-tag v1.2.3
```

### Переменные окружения

```bash
# Обязательные
export SERVER_IP=pinz.website
export POSTGRES_PASSWORD=your-db-password
export JWT_SECRET_KEY=your-jwt-secret

# Деплой
export IMAGE_TAG=v1.0.0
export SKIP_PULL=true              # пропустить docker pull (k3s)

# SMTP (auth-service, отправка кодов верификации)
export SMTP_HOST=smtp.example.com
export SMTP_PORT=587
export SMTP_USERNAME=noreply@pinz.website
export SMTP_PASSWORD=your-smtp-password
export SMTP_FROM=noreply@pinz.website

# S3 (trip-service, медиа)
export S3_ENDPOINT=https://storage.yandexcloud.net
export S3_REGION=ru-central1
export S3_BUCKET=pinz-media
export S3_ACCESS_KEY=your-access-key
export S3_SECRET_KEY=your-secret-key
export S3_PRESIGN_TTL=15m

# Geocoding (statistics-service, BigDataCloud reverse geocode).
# Trip-service публикует PIN_LOCATIONS_REQUESTED (pinz:stats:events) → statistics
# резолвит координаты и пришлёт PIN_LOCATIONS_RESOLVED в pinz:trip:geo_events
# для mirror'а в реплику trip-service.
export GEOCODING_BASE_URL=https://api.bigdatacloud.net/data/reverse-geocode-client  # optional, default shown
export GEOCODING_API_KEY=                                                     # optional, free tier works without key

# Share-link (api-gateway): база для поля share_url в ответах с трипом (ТЗ 3.4).
# Пустая → используется внутренний дефолт https://pinz.website/trips.
export TRIP_SHARE_LINK_BASE=https://pinz.website/trips

# Apple App ID (api-gateway) для /.well-known/apple-app-site-association.
# Формат <TeamID>.<BundleID>. Без него universal-links и Sign in with Apple
# работать не будут — встроенный дефолт это плейсхолдер.
export APPLE_APP_ID=ABCDE12345.com.example.pinz

# Loadtest-only: разблокирует RPC AuthService.DevLogin и проксирующую ручку
# POST /api/v1/auth/dev-login. По умолчанию выключено и НЕ должно включаться на
# проде — это путь обхода passkey. Используется только сидером нагрузочного
# тестирования (см. Backend/loadtest/).
export AUTH_DEV_LOGIN_ENABLED=false   # auth-service
export DEV_LOGIN_PROXY_ENABLED=false  # api-gateway-service

# ML-интеграция (trip-service). При false трип завершается синхронно, без ML.
export ML_ENABLED=true
```

### SSL/TLS (Let's Encrypt)

```bash
DOMAIN=pinz.website EMAIL=admin@pinz.website ./setup-cert-manager.sh
# → https://pinz.website доступен после выпуска сертификата

# С поддоменами Grafana и Kiali (один сертификат на все имена):
EXTRA_DOMAINS=grafana.pinz.website,kiali.pinz.website DOMAIN=pinz.website EMAIL=admin@pinz.website ./setup-cert-manager.sh

kubectl get secret pinz-tls -n istio-system
```

### CI/CD (GitHub Actions)

- **CI**: сборка, тесты, линтинг — на каждом PR/push
- **CD**: автодеплой при мерже в `main`

GitHub Secrets:

| Secret | Описание |
|---|---|
| `VPS_HOST` | IP/домен сервера |
| `VPS_USER` | SSH пользователь |
| `VPS_SSH_KEY` | Приватный SSH ключ |
| `POSTGRES_PASSWORD` | Пароль БД |
| `JWT_SECRET_KEY` | Секрет JWT |
| `DOCKER_REGISTRY` | Docker registry (e.g. ghcr.io) |
| `DOCKER_REPO` | Docker repository |
| `DOCKER_USERNAME` | Docker registry логин |
| `DOCKER_PASSWORD` | Docker registry пароль |
| `SERVER_IP` | IP сервера |

## Loadtest stand

Нагрузочное тестирование разворачивается на отдельном временном VPS (не на проде).

```bash
# На свежем VPS из-под root:
bash setup-server.sh --profile loadtest --branch <FEATURE_BRANCH>
```

Деталь по разворачиванию, сценариям и SLO — внутри `Backend/loadtest/` (k6-сценарии в `k6/scenarios/`, манифесты стабов в `manifests/`, Makefile с целями `seed`/`smoke`/`baseline`/`load`/`soak`/`cleanup`).

## Observability

Весь стек работает в k8s (namespace `default`). Приложения отправляют телеметрию на `otel-collector:4317` (OTLP gRPC). На VPS observability поднимается при деплое (`deploy.sh`).

**Что собирается:**

| Сигнал | Источник | Бэкенд |
|---|---|---|
| Трейсы | HTTP-запросы, gRPC-вызовы, SQL, Redis | Tempo |
| Метрики | RED (rate/errors/duration), auth counters, Go runtime | Prometheus |
| Логи | `slog` → OTLP; логи контейнеров с нод → Promtail | Loki |

**Grafana:**

| Среда | Доступ |
|---|---|
| Локально | `make grafana` → http://localhost:3000 |
| VPS (Production) | https://grafana.pinz.website |

**Корреляции в Grafana**: клик по трейсу открывает связанные логи и метрики.

**Kiali (service mesh):** граф сервисов, RPS/latency на edges, mTLS-статус, валидация Istio-конфига.

| Среда | Доступ |
|---|---|
| Локально | `make kiali` → http://localhost:20001 |
| VPS (Production) | https://kiali.pinz.website |

Kiali поднимает свой изолированный Prometheus в `istio-system` (для Envoy-метрик), а Grafana/Tempo тянет из стандартного observability-стека. Установка аддонов — разовая, через `make kiali-up` (локально) или `setup-server.sh` (на свежем VPS).

## Регенерация proto и swagger

```bash
# Proto — из корня Backend/
make proto

# Swagger
make swagger
```

Подробная документация: `DEPLOYMENT.md`, `ci-cd.md`, `deploy.md`.
