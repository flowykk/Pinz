# Skill: CI/CD для Pinz Backend в GitHub Actions

## 1. Goal

Документ описывает CI/CD-пайплайн для микросервисного бэкенда Pinz на базе GitHub Actions. Поддерживаются:

- **CI** — сборка, тесты, линтинг, проверка proto/swagger
- **CD** — публикация образов в Container Registry, деплой на VPS (опционально)

Документ предназначен для агента, который реализует workflows и конфигурации.

---

## 2. Архитектура пайплайна

### 2.1 Ветки и окружения

| Ветка | Назначение | Триггер | Действия |
|-------|-------------|---------|----------|
| `main` | Релизы | push, PR | CI (build, test, lint), CD (push image, deploy prod) |
| `develop` | Интеграция | push, PR | CI (build, test, lint), CD (push image, deploy staging) |
| `PINZ-<taskNumber>` | Задачи (например PINZ-123) | push, PR | CI (build, test, lint) |

### 2.2 Workflow-файлы

```
Pinz/
├── .github/
│   └── workflows/
│       ├── ci.yaml           # lint, test, build
│       ├── cd.yaml           # push images, deploy
│       └── proto-swagger.yaml # проверка proto/swagger при изменении
```

---

## 3. CI Workflow

### 3.1 Триггеры

- `push` на `main`, `develop`, `PINZ-*`
- `pull_request` на `main`, `develop`, `PINZ-*`
- Пути: `Backend/**`, `proto/**`, `.github/workflows/ci.yaml`

### 3.2 Шаги (где выполняется)

| Шаг | Runs on | Команды |
|-----|---------|---------|
| Checkout | runner | `actions/checkout@v4` |
| Setup Go | runner | `actions/setup-go@v5` с `go-version: '1.23'` |
| Cache | runner | `actions/cache` для `~/.cache/go-build`, `go.sum` |
| Lint | runner | `golangci-lint run ./...` (для каждого сервиса) |
| Test | runner | `go test ./...` (для каждого сервиса) |
| Build | runner | `docker build` для `api-gateway-service` и `auth-service` |
| Proto check | runner | при изменении `Backend/proto/*` — `make proto` и проверка, что сгенерированные файлы не изменились |
| Swagger check | runner | при изменении `api-gateway-service` — `make swagger` и проверка docs |

### 3.3 Матрица (опционально)

Запуск тестов для нескольких сервисов параллельно:

```yaml
strategy:
  matrix:
    service: [api-gateway-service, auth-service]
```

### 3.4 Пример структуры ci.yaml

```yaml
name: CI

on:
  push:
    branches: [main, develop, 'PINZ-*']
    paths:
      - 'Backend/**'
      - 'proto/**'
  pull_request:
    branches: [main, develop, 'PINZ-*']
    paths:
      - 'Backend/**'
      - 'proto/**'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          working-directory: Backend/api-gateway-service
      # аналогично для auth-service

  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [api-gateway-service, auth-service]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go test ./...
        working-directory: Backend/${{ matrix.service }}

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - run: docker build -t api-gateway-service:v${{ github.sha }} -f Backend/api-gateway-service/Dockerfile Backend/api-gateway-service
      - run: docker build -t auth-service:v${{ github.sha }} -f Backend/auth-service/Dockerfile Backend/auth-service
```

---

## 4. CD Workflow

### 4.1 Триггеры

- `push` на `main` (deploy prod, релизы)
- `push` на `develop` (deploy staging, опционально)
- `PINZ-*` — без деплоя
- Зависит от успешного CI или запускается после merge

### 4.2 Где выполняется

Все шаги CD — на `ubuntu-latest` runner, кроме деплоя на VPS (SSH).

### 4.3 Шаги CD

| Шаг | Runs on | Действия |
|-----|---------|----------|
| Checkout | runner | `actions/checkout@v4` |
| Login to Registry | runner | `docker/login-action` (GHCR, Docker Hub, или другой) |
| Build & Push | runner | build образов, push с тегами `latest`, `sha-<short>` |
| Deploy to VPS | runner | SSH на VPS, `kubectl apply` или `helmfile apply` |

### 4.4 Container Registry

Рекомендуется GitHub Container Registry (ghcr.io):

- `ghcr.io/<owner>/pinz-api-gateway:latest`
- `ghcr.io/<owner>/pinz-auth-service:latest`

Для push — `GITHUB_TOKEN` или персональный PAT с `write:packages`.

### 4.5 Secrets (GitHub)

| Secret | Использование |
|--------|---------------|
| `REGISTRY_TOKEN` / `GITHUB_TOKEN` | Push в ghcr.io |
| `VPS_HOST` | Hostname или IP VPS |
| `VPS_SSH_KEY` | Приватный ключ для SSH |
| `VPS_USER` | Пользователь SSH (например, `root` или `deploy`) |
| `KUBECONFIG` или `KUBECONFIG_BASE64` | kubeconfig для деплоя (если не через SSH) |

### 4.6 Деплой на VPS

**Вариант A: SSH + kubectl/helmfile на VPS**

```bash
ssh $VPS_USER@$VPS_HOST "
  cd /opt/pinz/Backend &&
  kubectl set image deployment/api-gateway api-gateway=ghcr.io/owner/pinz-api-gateway:$IMAGE_TAG &&
  kubectl set image deployment/auth-service auth-service=ghcr.io/owner/pinz-auth-service:$IMAGE_TAG
"
```

**Вариант B: SSH + pull + helmfile**

```bash
ssh $VPS_USER@$VPS_HOST "
  cd /opt/pinz/Backend &&
  helmfile apply --set image.tag=$IMAGE_TAG
"
```

**Вариант C: Self-hosted runner на VPS**

Runner зарегистрирован на VPS, имеет доступ к `kubectl` и `kubeconfig`. CD job выполняется на `self-hosted` runner.

### 4.7 Условный деплой

- `main` → deploy prod (только при push, не при PR)
- `develop` → deploy staging (опционально)
- `PINZ-*` → без деплоя

---

## 5. Proto и Swagger проверки

### 5.1 Отдельный workflow (опционально)

При изменении `Backend/proto/*.proto` или `api-gateway-service/**`:

1. Запустить `make proto` / `make swagger`
2. Проверить, что сгенерированные файлы не изменились (`git diff --exit-code`)

Если изменились — fail с сообщением: «Запусти make proto / make swagger и закоммить изменения».

### 5.2 Интеграция в CI

Можно включить в основной `ci.yaml` как job `proto-check` и `swagger-check` с `paths`-фильтром.

---

## 6. Артефакты для агента

Агент должен создать:

### 6.1 .github/workflows/ci.yaml

- Jobs: lint, test, build
- Path filters для Backend
- Branches: `main`, `develop`, `PINZ-*`
- Go 1.23, golangci-lint

### 6.2 .github/workflows/cd.yaml

- Trigger: push на `main` (и опционально `develop`)
- Jobs: build-and-push, deploy
- `needs: ci` или отдельный workflow с `workflow_run`

### 6.3 .github/workflows/proto-swagger.yaml (опционально)

- Trigger: изменение proto или swagger-аннотаций
- Check: регенерация и diff

### 6.4 Дополнительно

- `.golangci.yaml` в корне или в `Backend/` для линтера
- `Makefile` targets: `test`, `lint`, `build` (если ещё нет)

---

## 7. Сводка: где что выполняется

| Компонент | Где |
|-----------|-----|
| CI (lint, test, build) | GitHub-hosted runner (ubuntu-latest) |
| CD (build image, push) | GitHub-hosted runner |
| CD (deploy на VPS) | GitHub runner → SSH на VPS, или self-hosted runner на VPS |

---

## 8. Рекомендуемый порядок внедрения

1. **CI** — lint, test, build (без деплоя)
2. **CD** — push образов в ghcr.io
3. **CD** — деплой на VPS (SSH или self-hosted)
4. **Proto/Swagger** — проверка при PR

---

## 9. Переменные окружения для тестов

| Переменная | Значение в CI | Описание |
|------------|---------------|----------|
| `DB_HOST` | localhost или пропустить | Для интеграционных тестов (если есть) |
| `REDIS_ADDR` | localhost:6379 или пропустить | Аналогично |
| `JWT_SECRET_KEY` | test-secret | Для unit-тестов |

Если интеграционные тесты требуют PostgreSQL/Redis — использовать `services:` в workflow или `testcontainers`.
