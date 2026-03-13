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
| **api-gateway-service** | REST → gRPC прокси, HTTP-трейсинг |
| **auth-service** | Бизнес-логика авторизации, passkey, JWT |
| **otel-collector** | Приём и маршрутизация телеметрии (OTLP) |
| **tempo** | Хранение распределённых трейсов |
| **prometheus** | Метрики (RED + бизнес + Go runtime) |
| **loki** | Структурированные логи |
| **grafana** | Дашборды, корреляция трейс↔лог↔метрика |

## Стек

Go 1.25 · chi · gRPC · Protobuf · PostgreSQL · Redis · JWT · OpenTelemetry · Istio · Helm · Helmfile

## Эндпоинты (REST)

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/auth/email` | Ввод почты. Ответ: `{"is_registered": bool, "registration_key": "..."}` |
| POST | `/api/v1/auth/verify-email` | Проверка кода из письма |
| POST | `/api/v1/auth/finish-register` | Установка пароля и username |
| POST | `/api/v1/auth/login` | Вход по email + пароль |
| POST | `/api/v1/auth/refresh` | Обновление access-токена |
| POST | `/api/v1/auth/logout` | Выход |

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

Весь стек работает в k8s (namespace `default`). Приложения отправляют телеметрию на `otel-collector:4317` (OTLP gRPC).

**Что собирается:**

| Сигнал | Источник | Бэкенд |
|---|---|---|
| Трейсы | HTTP-запросы, gRPC-вызовы, SQL, Redis | Tempo |
| Метрики | RED (rate/errors/duration), auth counters, Go runtime | Prometheus |
| Логи | `slog` → OTLP bridge | Loki |

**Корреляции в Grafana**: клик по трейсу открывает связанные логи и метрики.

## Регенерация proto и swagger

```bash
# Proto — из корня Backend/
make proto

# Swagger
make swagger
```

Подробная документация: `DEPLOYMENT.md`, `ci-cd.md`, `deploy.md`.
