SHELL := /bin/bash

COMPOSE      := docker compose
COMPOSE_INFRA := docker compose -f docker-compose.infra.yml
VENV         := .venv
PYTHON       := $(VENV)/bin/python
PYTEST       := $(VENV)/bin/pytest

.PHONY: help setup proto export-models \
        infra-up infra-down infra-no-triton triton-up \
        dev-up dev-down \
        build up all-up down logs ps \
        test test-unit test-integration test-slow lint smoke \
        clean

help:
	@echo ""
	@echo "=== Локальная разработка ==="
	@echo "  setup              - создать venv (Python 3.11) + установить все зависимости"
	@echo "  proto              - сгенерировать Python gRPC стабы из *.proto"
	@echo "  infra-up           - docker: NATS + Qdrant + Meilisearch + Triton-stub"
	@echo "  infra-no-triton    - docker: NATS + Qdrant + Meilisearch (без Triton)"
	@echo "  dev-up             - запустить все 6 сервисов нативно в фоне"
	@echo "  dev-down           - остановить нативные сервисы"
	@echo "  smoke              - проверить что все сервисы живы (health-check)"
	@echo ""
	@echo "=== Тесты ==="
	@echo "  test               - все unit-тесты (без инфры, быстро)"
	@echo "  test-unit          - только unit/ (псевдоним для test)"
	@echo "  test-integration   - e2e тесты (требует infra-up + dev-up)"
	@echo "  test-slow          - тесты с реальной загрузкой моделей (медленно)"
	@echo ""
	@echo "=== Docker (production-like) ==="
	@echo "  build              - собрать все Docker образы"
	@echo "  up                 - запустить все сервисы в Docker"
	@echo "  all-up             - infra + build + up"
	@echo "  down               - остановить Docker сервисы"
	@echo "  logs S=<service>   - хвост логов сервиса"
	@echo ""
	@echo "  export-models      - экспорт HF моделей → ONNX (нужен torch+transformers)"
	@echo "  clean              - удалить proto/gen и ONNX веса"
	@echo ""

# -----------------------------------------------------------------------
# Локальная разработка
# -----------------------------------------------------------------------

setup:
	bash scripts/setup_venv.sh

proto:
	bash scripts/gen_proto.sh

export-models:
	$(PYTHON) triton/scripts/export_models.py

infra-up:
	$(COMPOSE_INFRA) up -d

infra-no-triton:
	$(COMPOSE_INFRA) up -d nats qdrant meilisearch

infra-down:
	$(COMPOSE_INFRA) down

triton-up:
	$(COMPOSE_INFRA) up -d triton

dev-up:
	bash scripts/run_local.sh

dev-down:
	bash scripts/run_local.sh stop

smoke:
	$(PYTHON) scripts/smoke_test.py

# -----------------------------------------------------------------------
# Тесты
# -----------------------------------------------------------------------

test: test-unit

test-unit:
	$(PYTEST) tests/unit/ -m "not slow" --tb=short

test-integration:
	$(PYTEST) tests/integration/ --tb=short

test-slow:
	$(PYTEST) -m slow --tb=short

lint:
	$(VENV)/bin/ruff check . --fix

# -----------------------------------------------------------------------
# Docker (production-like)
# -----------------------------------------------------------------------

build: proto
	$(COMPOSE) build

up:
	$(COMPOSE) up -d ml_orchestrator moderation_service embedding_service \
	  classifier_service similarity_service search_service

all-up: infra-up build up

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f $${S}

ps:
	$(COMPOSE) ps

# -----------------------------------------------------------------------

clean:
	rm -rf proto/gen
	find triton/model_repository -name "*.onnx" -delete
	find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
