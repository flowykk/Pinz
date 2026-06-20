# Pinz

Оживи свой маршрут. Запомни каждое впечатление. Запинь каждое мгновение своего путешествия!

> **Основная ветка разработки — [`develop`](https://github.com).**  
> Ветка `main` — для production-деплоя бэкенда. PR и ежедневная работа — в `develop`.

Монорепозиторий: **iOS-клиент** и **Go-бэкенд** (микросервисы, Kubernetes, CI).

---

## Скриншоты

<img width="485" height="1023" alt="Без заголовка" src="https://github.com/user-attachments/assets/3a40ebdd-f852-4b53-9abc-ad34a1d36097" />

<img width="473" height="1023" alt="1" src="https://github.com/user-attachments/assets/baf4e512-b77f-43fa-8521-e368aa693816" />



## Структура репозитория

```
.
├── Pinz/                 # iOS-приложение (Tuist, SwiftUI, iOS 18+)
│   └── Modules/          # Модули приложения (Domain, Networking, Trips, …)
├── Backend/              # Микросервисный бэкенд (Go, gRPC, PostgreSQL, Redis)
├── .github/workflows/    # CI/CD
└── README.md
```

| Часть | Описание | Документация |
|-------|----------|--------------|
| [`Pinz/`](Pinz/) | iOS: путешествия, пины, карта, лента, passkey, push (APNs) | [`Pinz/README.md`](Pinz/README.md) |
| [`Backend/`](Backend/) | API Gateway, auth, trip, statistics, notification | [`Backend/README.md`](Backend/README.md) |

---

## Быстрый старт

### iOS (`Pinz/`)

**Нужно:** macOS, Xcode 16+, [Tuist](https://docs.tuist.io).

```bash
cd Pinz
tuist install
tuist generate
open Pinz.xcworkspace
```

- Bundle ID: `io.tuist.hse.Pinz`
- API prod: `https://pinz.website`
- Локальный API: аргумент запуска `-useLocalhost` → `http://localhost:8080`

Push — только на **реальном устройстве**. Подробнее: [`Pinz/README.md`](Pinz/README.md).

### Backend (`Backend/`)

**Нужно:** Docker; для полного стека — Minikube, kubectl, Helm, Helmfile, istioctl.

```bash
cd Backend
make infra-up
make dev          # → http://localhost:8080
```

Полный стек: `make all-up`. Swagger: `http://localhost:8080/swagger/index.html`.

Подробнее: [`Backend/README.md`](Backend/README.md).

---

## Клиент ↔ сервер

- REST + JWT через **api-gateway** (`/api/v1/...`)
- WebSocket — creation trip, pin-upload, add media
- Push: iOS → `POST /api/v1/profile/device-tokens`; отправка — **notification-service** (APNS-секреты в GitHub Actions)

---

## CI/CD

- **CI** (`ci-*`, `sqlc`, `proto-swagger`) — проверки на PR
- **CD** (`cd.yaml`) — деплой при push в `main` / `develop` с изменениями в `Backend/**`

Секреты APNS и др.: GitHub → Settings → Secrets → Actions (см. `cd.yaml`).

---

## Ветки

| Ветка | Назначение |
|-------|------------|
| **`develop`** | Основная ветка разработки |
| `main` | Production / деплой бэкенда |
| `PINZ-*` | Feature-ветки по задачам |

---

## См. также

- [`Pinz/README.md`](Pinz/README.md) — модули, MVVM, навигация
- [`Backend/README.md`](Backend/README.md) — архитектура, API, `make`, k8s
- [`Backend/loadtest/manifests/README.md`](Backend/loadtest/manifests/README.md)
