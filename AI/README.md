# Pinz AI Module

Микросервисный ML-модуль приложения Pinz. Реализован на Python, общается с Go-бэкендом через
NATS JetStream.

## Функционал

| # | Задача | Сервис | Модель |
|---|--------|--------|--------|
| 1 | NSFW-модерация изображений | Moderation | `Falconsai/nsfw_image_detection` |
| 1 | Токсичность текста (ru/en) | Moderation | `textdetox/xlmr-large-toxicity-classifier` (multilingual) |
| 2 | Классификация пина (6 категорий) | Classifier | `SigLIP` ViT-B/16 emb + MLP-голова |
| 3 | Похожие изображения внутри пина | Similarity | `pHash` → `SigLIP` cosine → Union-Find |
| 4 | Поиск пинов по тексту | Search | Hybrid: `SigLIP` text → Qdrant + Meilisearch (BM25) → RRF |
| — | Эмбеддинги изображений и текстов | Embedding | `SigLIP` ViT-B/16 (через Triton) |

## Архитектура

```
                 Pinz Backend (Go)
                       │
                  NATS JetStream
                       │
                       ▼
┌──────────────── ML Orchestrator ────────────────┐
│  (Python, asyncio, pull-consumer "ml-workers")  │
│                                                 │
│   gRPC ↓        gRPC ↓        gRPC ↓     gRPC ↓ │
│ Moderation  Embedding   Classifier  Similarity  │
│                  │                              │
│                  │ upsert ▼                     │
│              ┌──────────┐                       │
│              │  Qdrant  │  ◀── Search Service   │
│              └──────────┘                       │
│              ┌──────────────┐                   │
│              │ Meilisearch  │ ◀── Search Service│
│              └──────────────┘                   │
│                                                 │
│   ⇧ inference (CLIP/SigLIP/NSFW) [gRPC HTTP/2]  │
│                                                 │
│          NVIDIA Triton Inference Server         │
└─────────────────────────────────────────────────┘
```

## Структура репозитория

```
AI/
├── proto/                      # gRPC-контракты между сервисами
├── shared/                     # общая Python-библиотека (NATS, S3, контракты, утилиты)
├── ml_orchestrator/            # точка входа из NATS, оркестрация подзадач
├── moderation_service/         # NSFW + toxicity
├── embedding_service/          # SigLIP embeddings + Qdrant upsert
├── classifier_service/         # классификация пинов
├── similarity_service/         # детектор похожих изображений
├── search_service/             # гибридный поиск
├── triton/                     # model_repository и скрипты экспорта
├── deploy/                     # k8s-манифесты
├── scripts/                    # вспомогательные скрипты
├── docker-compose.yml          # локальный запуск всего стека
├── docker-compose.infra.yml    # только инфраструктура (NATS, Qdrant, Meili)
└── Makefile
```

## Быстрый старт (локально)

```bash
# 1. Экспорт моделей в model_repository (один раз, требует Python + torch на хосте)
make export-models

# 2. Поднять инфру + Triton
make infra-up
make triton-up

# 3. Поднять все сервисы
make up

# 4. Логи оркестратора
make logs S=ml_orchestrator
```

Для подключения к продовой шине (`wss://nats.pinz.website`) нужно поставить `NATS_URL` и
`NATS_TOKEN` в `.env`. Для разработки используется локальный `nats://nats:4222`.
