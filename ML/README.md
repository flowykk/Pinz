# Pinz AI Module — Research & Prototypes

AI-модуль Pinz обрабатывает медиа и текст пользователей: обнаруживает дубликаты фотографий, модерирует контент, классифицирует пины и обеспечивает семантический поиск. Модуль работает асинхронно — получает задачи из NATS JetStream и возвращает результаты серверной части приложения.


---

## Содержание

- [Контекст](#контекст)
- [Архитектура исследования](#архитектура-исследования)
- [Ноутбуки](#ноутбуки)
- [Прототипы сервисов](#прототипы-сервисов)
- [Библиотека pinz-ml](#библиотека-pinz-ml)
- [Структура проекта](#структура-проекта)
- [Быстрый старт](#быстрый-старт)
- [Итоги и переход в продакшн](#итоги-и-переход-в-продакшн)

---

## Контекст

AI-разработка Pinz. Здесь решались три задачи:

1. **Выбор моделей** — сравнительный анализ кандидатов по качеству, скорости и размеру под реальные данные приложения.
2. **Обучение** — дообучение финальных моделей на специфике пользовательского контента.
3. **Прототипирование** — минимальные сервисы для валидации архитектурных решений до написания продакшн-кода.

---

## Архитектура

```
                         ┌─────────────────────────────────────┐
  Pinz Backend           │            AI Module (GPU Server)   │
  ──────────────         │                                     │
  trip-service           │  ┌──────────────────────────────┐   │
      │                  │  │       ml_orchestrator        │   │
      │  ml.tasks.*      │  │  (NATS consumer + gRPC fan)  │   │
      └──────────────────┼─▶│                              │   │
                         │  └──┬──────┬──────┬──────┬─────┘   │
  trip-worker            │     │      │      │      │          │
      ▲                  │     ▼      ▼      ▼      ▼          │
      │  ml.results.*    │  embed  moder  class  simil  search │
      └──────────────────┼─ ding   ation  ifier  arity  _svc   │
                         │  :52   :51    :53    :54    :55     │
                         │     │      │                        │
  wss://nats.pinz.website│     └──────┴──── Triton :8001 ─────┤
                         │                 (SigLIP + NSFW)    │
                         │                                     │
                         │         Qdrant :6333                │
                         │         Meilisearch :7700           │
                         └─────────────────────────────────────┘
```

**Ключевые решения:**
- **Pull-модель NATS JetStream** — `ml_orchestrator` сам выбирает темп через `Fetch(N)`, задачи не теряются при перезапуске.
- **Внутренняя шина — gRPC** — все 5 сервисов общаются через строго типизированные Protocol Buffers.
- **GPU-инференс через Triton** — тяжёлые модели (SigLIP, NSFW) обслуживаются как ONNX-сессии с dynamic batching.
- **Изоляция** — сервис не принимает входящих соединений, только исходящий NATS (443/wss) и локальный gRPC.

---

## Ноутбуки

### `notebooks/embeddings/` — выбор модели эмбеддингов

| Ноутбук | Что исследовалось |
|---|---|
| `clip_image_embeddings.ipynb` | CLIP ViT-L/14 (OpenAI) |
| `embedding_benchmark.ipynb` | Сравнение CLIP vs SigLIP SO400M|
| `multilingual_e5_large.ipynb` | `intfloat/multilingual-e5-large` — RU/EN качество на тревел-описаниях |

---

### `notebooks/image_moderation/` — NSFW-детекция изображений

| Ноутбук | Модель | Примечание |
|---|---|---|
| `opennsfw2.ipynb` | OpenNSFW2 (ResNet-50) | Базовый бенчмарк |
| `resnet50.ipynb` | ResNet-50 fine-tune | Дообучение на NSFW датасете |
| `marqo_nsfw_image_detection_384.ipynb` | `marqo/nsfw-image-detection-384` | ViT-384, быстрый |
| `falconsai_nsfw_image_detection.ipynb` | `Falconsai/nsfw_image_detection` | HuggingFace-модель |
| `clip_zero_shot_vit_l14.ipynb` | CLIP ViT-L/14 zero-shot | Zero-shot без дообучения |

---

### `notebooks/text_moderation/` — токсичность текста

| Ноутбук | Модель | Языки |
|---|---|---|
| `cointegrated_rubert_tiny_toxicity.ipynb` | `cointegrated/rubert-tiny-toxicity` | RU, 28M params |
| `s_nlp_russian_toxicity_classifier.ipynb` | `s-nlp/russian_toxicity_classifier` | RU |
| `multilingual_toxic_xlm_roberta.ipynb` | `unitary/multilingual-toxic-xlm-roberta` | RU/EN/… |
| `unitary_toxic_bert.ipynb` | `unitary/toxic-bert` | EN |
| `llama_guard_8b.ipynb` | Meta LLaMA Guard 2 8B (vLLM) | EN, много категорий |

---

### `notebooks/pin_classification/` — классификация пинов

| Ноутбук | Подход | Точность |
|---|---|---|
| `pin_zero_shot_clip_vit_l14.ipynb` | Zero-shot CLIP |
| `pin_zero_shot_siglip_so400m.ipynb` | Zero-shot SigLIP |
| `pin_places365_resnet50.ipynb` | Places365 transfer |
| `pin_efficientnet_b4_finetune.ipynb` | EfficientNet-B4 fine-tune |
| `pin_siglip_mlp_img_only.ipynb` | SigLIP + MLP head |

---

### `notebooks/search/` — поиск пинов

| Ноутбук | Подход |
|---|---|
| `elasticsearch_pins.ipynb` | BM25 (Elasticsearch), full-text |
| `qdrant_semantic_search.ipynb` | Векторный поиск (Qdrant, COSINE) |
| `hybrid_search_rrf.ipynb` | RRF-слияние BM25 + vector |

---

### `notebooks/inference/` — сравнение inference-движков

| Ноутбук | Что тестировалось |
|---|---|
| `vllm_llama_guard.ipynb` | LLaMA Guard 2 8B через vLLM |
| `sglang_benchmark.ipynb` | SGLang vs vLLM: latency/throughput |

---

## NATS-контракт

Модуль читает задачи из стрима `ML_TASKS` (consumer `ml-workers`) и публикует в `ML_RESULTS`. Retry и DLQ полностью на стороне бэкенда (`MaxDeliver=5`, `AckWait=10m`).

### Входящие flows (`ml.tasks.<flow>`)

| Flow | Описание |
|---|---|
| `creation` | Новый трип, набор новых пинов с медиа |
| `add_media` | Добавление медиа в существующий трип |
| `pin_upload.creation` | Создание нового пина через upload |
| `pin_upload.addition` | Добавление к существующему пину |
| `text_moderation` | Модерация текстовых полей трипа/пинов |

### Исходящие результаты (`ml.results.<flow>`)

```json
{
  "flow": "creation",
  "trip_id": "uuid",
  "similar_groups": [["media_id_1", "media_id_2"]],
  "nsfw_ids": ["media_id_3"],
  "pin_suggestions": [
    { "pin_id": "uuid", "category": "food", "tags": ["restaurant"] }
  ],
  "text_results": []
}
```

---

## Структура проекта

```
ML/
├── notebooks/
│   ├── embeddings/             # Бенчмарк моделей эмбеддингов
│   ├── image_moderation/       # Сравнение NSFW-детекторов
│   ├── text_moderation/        # Сравнение токсичность-классификаторов
│   ├── pin_classification/     # Эксперименты с классификацией пинов
│   ├── search/                 # BM25 vs vector vs hybrid
│   ├── inference/              # vLLM / SGLang benchmarks
│   ├── explore_images.ipynb    # Общий EDA изображений
│   ├── explore_text.ipynb      # Общий EDA текста
│   └── training_moderation.ipynb  # Обучение модерационных моделей
│
├── data/
│   ├── image_moderation/       # EDA-ноутбуки по датасетам (NSFW, Jigsaw)
│   ├── pin_classification/     # EDA Food101, iNaturalist, Places365
│   ├── text_moderation/        # EDA Russian Toxic, RuToxic
│   └── generate_synthetic.py   # Генератор синтетики
│
├── services/
│   ├── moderation_service/     # OpenNSFW2 + rubert-tiny 
│   ├── embedding_service/      # CLIP ViT-L/14 + E5-large 
│   ├── search_service/         # ES BM25 + Qdrant + RRF 
│   ├── vllm_service/           # GPU
│   ├── pin_classifier_service/ # EfficientNet-B4 fine-tune 
│   ├── trip_classifier_service/# TF-IDF+LogReg / zero-shot
│   └── api_gateway_service/    # gRPC stub
│       └── proto/
│           └── moderation.proto
│
├── src/
│   └── pinz_ml/
│       ├── paths.py
│       └── data/
│           └── parquet_loaders.py
│
├── docker-compose.yml          # Полный стек: ES + Qdrant + все сервисы
├── pyproject.toml              # Poetry: torch, transformers, fastapi, sklearn…
└── Makefile                    # install / notebooks / proto / run-* / compose-*
```

---

## Быстрый старт

**Требования:** Python 3.10+, Poetry, Docker.

```bash
# Установить зависимости
make install        # = poetry install

# Открыть ноутбуки
make notebooks      # JupyterLab на localhost:8888

# Поднять всю инфраструктуру + прототипы сервисов
make compose-up

# Запустить отдельный сервис (без Docker)
make run-moderation       # :8001
make run-search           # :8003
make run-trip-classifier  # :8006

# Сгенерировать gRPC-код из proto
make proto
```

---
