# Pinz Backend

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
| **otel-collector** | Приём и маршрутизация телеметрии (OTLP) |
| **tempo** | Хранение распределённых трейсов |
| **prometheus** | Метрики (RED + бизнес + Go runtime) |
| **loki** | Структурированные логи |
| **grafana** | Дашборды, корреляция трейс↔лог↔метрика |

## Стек

Go 1.25 · chi · gRPC · Protobuf · PostgreSQL · Redis · JWT · OpenTelemetry · Istio · Helm · Helmfile

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
| 1 | POST | `/api/v1/trips/creation/start` | Создание путешествия + выдача `files_to_upload` → S3 presigned URLs (пока URL заглушка) |
| 2 | POST | `/api/v1/trips/creation/{trip_id}/media/process-grouping` | Передача метаданных медиа, первичная группировка по гео/времени, статус `DRAFT_GROUPING_REVIEW` |
| 3 | POST | `/api/v1/trips/creation/{trip_id}/apply-groups-and-process` | Применение ручных групп, создание пинов, статус `PROCESSING`, задача в Redis Streams `pinz:trip:ml:tasks` |
| 4 | GET | `/api/v1/trips/creation/{trip_id}/review` | Результат фоновой обработки: пины, теги, `issues`, массив `similar` (похожие медиа) |
| 5 | POST | `/api/v1/trips/creation/{trip_id}/finalize` | Финальное ревью, ручные правки, удаление медиа, агрегация трипа, статус `READY` |

Real‑time уведомление о завершении шага 3–4 идёт через WebSocket:

| Метод | Путь | Описание |
|---|---|---|
| GET | `/v1/ws` | WebSocket‑канал. Аутентификация по JWT; сервер подписывает подключение на Redis Pub/Sub `pinz:user:{user_id}:events`. Воркер trip-service публикует событие `TRIP_PROCESSING_COMPLETED` с payload `{ "trip_id", "status": "DRAFT_FINAL_REVIEW" }`. |

### Trip core / feed (trip-service через API Gateway)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/trips` | Список своих путешествий (учитывая участие в трипах) |
| POST | `/api/v1/trips` | Создание путешествия (тот же gRPC, что и `trips/creation/start`; основной путь — `trips/creation/start`) |
| GET | `/api/v1/trips/{id}` | Детали путешествия (только участники или те, у кого в избранном при soft‑delete) |
| PATCH | `/api/v1/trips/{id}` | Редактирование параметров путешествия (название, сезон, приватность и т.д.) |
| DELETE | `/api/v1/trips/{id}` | Удаление путешествия с учётом избранного (полное или soft‑delete; см. PINZ‑98) |
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
# Инфраструктура (PostgreSQL, Redis) — Docker Compose
make infra-up

# Observability (OTel Collector, Tempo, Prometheus, Loki, Grafana) — k8s
make obs-up

# Сборка образов внутри Minikube и деплой приложения
make k8s-build
make k8s-istio
make k8s-deploy

# Или всё одной командой
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
make obs-up         # Запустить стек (OTel Collector, Tempo, Prometheus, Loki, Grafana)
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
export IMAGE_TAG=v1.0.0
export SERVER_IP=pinz.website
export POSTGRES_PASSWORD=your-db-password
export JWT_SECRET_KEY=your-jwt-secret
```

### SSL/TLS (Let's Encrypt)

```bash
DOMAIN=pinz.website EMAIL=admin@pinz.website ./setup-cert-manager.sh
# → https://pinz.website доступен после выпуска сертификата

# С поддоменом для Grafana (один сертификат на оба имени):
EXTRA_DOMAINS=grafana.pinz.website DOMAIN=pinz.website EMAIL=admin@pinz.website ./setup-cert-manager.sh

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

## Observability

Весь стек работает в k8s (namespace `default`). Приложения отправляют телеметрию на `otel-collector:4317` (OTLP gRPC). На VPS стек observability (OTel Collector, Tempo, Prometheus, Loki, Grafana) поднимается автоматически при деплое (`deploy.sh`).

**Что собирается:**

| Сигнал | Источник | Бэкенд |
|---|---|---|
| Трейсы | HTTP-запросы, gRPC-вызовы, SQL, Redis | Tempo |
| Метрики | RED (rate/errors/duration), auth counters, Go runtime | Prometheus |
| Логи | `slog` → OTLP bridge | Loki |

**Grafana:**

| Среда | Доступ |
|---|---|
| Локально | `make grafana` → http://localhost:3000 |
| VPS (Production) | https://grafana.pinz.website |

**Корреляции в Grafana**: клик по трейсу открывает связанные логи и метрики.

## Регенерация proto и swagger

```bash
# Proto — из корня Backend/
make proto

# Swagger
make swagger
```

Подробная документация: `DEPLOYMENT.md`, `ci-cd.md`, `deploy.md`.
