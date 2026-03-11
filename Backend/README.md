# Pinz Backend

Микросервисный бэкенд приложения Pinz (итерация 1: фундамент и авторизация).

## Запуск

### Локальная разработка
```bash
docker-compose up -d
```

### Production развертывание

**Требования:**
- Kubernetes кластер (k3s/minikube/kubeadm)
- Istio service mesh
- Helm, Helmfile, kubectl
- Docker registry доступ

**Развертывание:**
```bash
# На сервере в директории Backend
./deploy.sh

# Или с конкретным тегом
./deploy.sh --image-tag v1.0.0
```

**Окружение:**
```bash
# Переменные окружения (или .env файл)
export IMAGE_TAG=v1.0.0
export SERVER_IP=your-server-ip
export POSTGRES_PASSWORD=your-db-password
export JWT_SECRET_KEY=your-jwt-secret
```

**Адреса после запуска:**

| Окружение | Ресурс | URL / Команда |
|-----------|--------|---------------|
| **Локально** | API Gateway | http://localhost:8080 |
| | Swagger UI | http://localhost:8080/swagger/index.html |
| | Auth gRPC | localhost:50051 |
| | Логи | `docker-compose logs -f api-gateway-service` |
| **Production** | API Gateway | `http://<server-ip>:<port>` (автоматически определяется) |
| | Health check | `curl http://<server-ip>:<port>/health` |
| | Swagger UI | `http://<server-ip>:<port>/swagger/index.html` |
| | **Порт** | 30569 (NodePort) или 80 (LoadBalancer) |

## Эндпоинты (REST)

- `POST /api/v1/auth/email` — первый шаг: пользователь вводит почту. Тело: `{"email":"user@example.com"}`. Ответ: `{"is_registered": true|false, "registration_key": "..."}`. При `is_registered: true` клиент показывает экран ввода пароля (далее login); при `false` — экран ввода кода с почты (`registration_key` передаётся в verify-email и finish-register).
- `POST /api/v1/auth/verify-email` — проверка кода (при регистрации).
- `POST /api/v1/auth/finish-register` — установка пароля и username (при регистрации).
- `POST /api/v1/auth/login` — вход по email и паролю.
- `POST /api/v1/auth/refresh` — обновление access-токена.
- `POST /api/v1/auth/logout` — выход.

## Стек

- Go 1.23, chi, gRPC, Protobuf, PostgreSQL, Redis, Squirrel, JWT, swaggo.

## Регенерация proto и swagger

```bash
# Proto — единая точка из корня Backend (Backend/proto/*.proto)
cd Backend && make proto

# Swagger — только для API Gateway
cd api-gateway-service && make swagger
```

## Локальный линтинг

```bash
# Проверить все сервисы
make lint

# Проверить только API Gateway
make lint-api

# Проверить только Auth Service
make lint-auth
```

## Локальный деплой (Minikube + Istio)

Требуется: Minikube, Helm, Helmfile, istioctl, Docker.

```bash
# 1. Инфраструктура (PostgreSQL, Redis)
make infra-up

# 2. Minikube + Istio (выполнить один раз)
minikube start --cpus 4 --memory 4096 --driver=docker
minikube addons enable metrics-server
istioctl install --set profile=demo -y
kubectl label namespace default istio-injection=enabled

# 3. Istio-ресурсы (mTLS, Gateway, VirtualService)
make k8s-istio

# 4. Сборка образов и деплой приложения
make k8s-build
make k8s-deploy

# 5. Доступ (в отдельном терминале)
sudo minikube tunnel

# 6. Проброс хоста в /etc/hosts (один раз)
sudo sed -i '' '/pinz.example.com/d' /etc/hosts
echo "127.0.0.1 pinz.example.com" | sudo tee -a /etc/hosts
```

**Адреса (Minikube + Istio):**

| Ресурс | URL / Команда |
|--------|---------------|
| API Gateway | http://pinz.example.com |
| Swagger UI | http://pinz.example.com/swagger/index.html |
| Логи | `kubectl logs -f deployment/api-gateway` / `kubectl logs -f deployment/auth-service` |
| Трейсинг (Jaeger) | `istioctl dashboard jaeger` или `kubectl port-forward svc/tracing 16686:80 -n istio-system` → http://localhost:16686 |
| Мониторинг (Kiali) | `istioctl dashboard kiali` или `kubectl port-forward svc/kiali 20001:20001 -n istio-system` → http://localhost:20001 |
| Prometheus | `kubectl port-forward svc/prometheus 9090:9090 -n istio-system` → http://localhost:9090 |

Observability addons (Prometheus, Kiali, Jaeger) устанавливаются отдельно: см. `deploy.md`, раздел 9.

## Production развертывание (VPS)

### Полная настройка сервера

Используйте скрипт `setup-server.sh` для полной автоматизированной настройки:

```bash
# На чистом сервере Ubuntu 22.04
wget https://raw.githubusercontent.com/flowykk/Pinz/main/Backend/setup-server.sh
chmod +x setup-server.sh
./setup-server.sh --repo-url https://github.com/flowykk/Pinz.git
```

Этот скрипт установит:
- Docker, k3s, Helm, Helmfile, Istio
- Склонирует репозиторий
- Настроит инфраструктуру (PostgreSQL, Redis)
- Создаст переменные окружения
- Подготовит CI/CD доступ

### После настройки

```bash
cd /opt/pinz/Backend

# Ручной деплой
./deploy.sh

# Проверка статуса
kubectl get pods
curl http://<server-ip>:8080/health
```

### SSL/TLS сертификаты

Для HTTPS доступа настройте Let's Encrypt сертификаты:

```bash
cd /opt/pinz/Backend

# Установите сертификат для домена
DOMAIN=your-domain.com EMAIL=admin@your-domain.com ./setup-certbot.sh

# Проверьте что TLS secret создан
kubectl get secret pinz-tls -n istio-system

# Теперь доступен HTTPS: https://your-domain.com
```

Скрипт автоматически:
- Выпустит сертификат через Let's Encrypt
- Создаст Istio TLS secret
- Настроит автоматическое обновление
- Временно снимет редирект `80 -> Istio NodePort` на время проверки домена

**Требования:**
- Зарегистрированный домен указывающий на сервер
- Открытый входящий порт `80/tcp`
- Email для уведомлений Let's Encrypt

### Make команды (теперь работают)

```bash
# Управление инфраструктурой
make infra-up      # Запуск PostgreSQL + Redis
make infra-down    # Остановка инфраструктуры

# Kubernetes операции
make k8s-build     # Сборка Docker образов
make k8s-deploy    # Деплой в Kubernetes
make k8s-istio     # Установка Istio ресурсов

# Кодогенерация
make proto         # Генерация protobuf
make swagger       # Генерация swagger
```

### CI/CD

Проект поддерживает автоматическое развертывание через GitHub Actions:

- **CI**: Сборка, тесты, линтинг на каждом PR/push
- **CD**: Автоматический деплой на `main` ветку

Для настройки CD добавьте в GitHub Secrets:
- `VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`
- `POSTGRES_PASSWORD`, `JWT_SECRET_KEY`

### Ручной деплой с локальной машины

```bash
# Настройка SSH доступа к серверу
ssh-copy-id user@your-server

# Деплой с локальной машины
IMAGE_TAG=v1.2.3 ./deploy.sh
```

Подробная документация: `DEPLOYMENT.md`, `ci-cd.md`, `deploy.md`.
