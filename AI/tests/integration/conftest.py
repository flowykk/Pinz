"""Конфигурация интеграционных тестов.

Требования:
  - docker compose -f docker-compose.infra.yml up -d  (NATS, Qdrant, Meilisearch)
  - Все 6 сервисов запущены нативно в venv (scripts/run_local.sh)

Тесты с маркой @pytest.mark.integration автоматически пропускаются,
если NATS недоступен (SKIP_IF_NO_NATS=true).
"""

from __future__ import annotations

import asyncio
import os

import pytest


def _nats_available() -> bool:
    import socket
    try:
        s = socket.create_connection(("localhost", 4222), timeout=2)
        s.close()
        return True
    except OSError:
        return False


def pytest_collection_modifyitems(items):
    if not _nats_available():
        skip = pytest.mark.skip(reason="NATS not running (run: make infra-up)")
        for item in items:
            if "integration" in str(item.fspath):
                item.add_marker(skip)
