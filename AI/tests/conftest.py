"""Общие pytest-фикстуры для unit и integration тестов."""

from __future__ import annotations

import os
import sys
from pathlib import Path

import pytest

# ---------------------------------------------------------------------------
# Заблокировать AI/triton/ (Triton model repository) от импорта как Python-
# пакет 'triton'. Без этого блока torch._dynamo.has_triton_package() находит
# namespace-пакет из нашей директории и падает на triton.language.dtype.
# sys.modules[key] = None заставляет «import triton» бросать ImportError.
# ---------------------------------------------------------------------------
if "triton" not in sys.modules:
    sys.modules["triton"] = None  # type: ignore[assignment]

AI_ROOT = Path(__file__).resolve().parents[1]
PROTO_GEN = AI_ROOT / "proto" / "gen"
SHARED_SRC = AI_ROOT / "shared" / "src"

for p in (str(AI_ROOT), str(PROTO_GEN), str(SHARED_SRC)):
    if p not in sys.path:
        sys.path.insert(0, p)

for svc in (
    "ml_orchestrator",
    "moderation_service",
    "embedding_service",
    "classifier_service",
    "similarity_service",
    "search_service",
):
    src = str(AI_ROOT / svc / "src")
    if src not in sys.path:
        sys.path.insert(0, src)

# ---------------------------------------------------------------------------
# Если машина стоит за корпоративным прокси с самоподписанным сертификатом,
# huggingface_hub не может скачать модели (SSL handshake failure). Когда
# переменная HF_HUB_NO_VERIFY_SSL=1, подменяем дефолтную httpx-фабрику на
# клиент с verify=False. Включается только явно — не влияет на остальные тесты.
# ---------------------------------------------------------------------------
if os.environ.get("HF_HUB_NO_VERIFY_SSL") == "1":
    import httpx as _httpx
    from huggingface_hub.utils._http import set_client_factory as _set_hf_client

    def _no_verify_client() -> "_httpx.Client":
        return _httpx.Client(follow_redirects=True, timeout=None, verify=False)

    _set_hf_client(_no_verify_client)

# ---------- ENV ---------------------------------------------------------
os.environ.setdefault("NATS_URL", "nats://localhost:4222")
os.environ.setdefault("NATS_TOKEN", "")
os.environ.setdefault("QDRANT_URL", "http://localhost:6333")
os.environ.setdefault("MEILI_URL", "http://localhost:7700")
os.environ.setdefault("MEILI_MASTER_KEY", "pinz-dev-master-key")
os.environ.setdefault("TRITON_GRPC_URL", "localhost:8001")
os.environ.setdefault("MODERATION_GRPC", "localhost:50051")
os.environ.setdefault("EMBEDDING_GRPC", "localhost:50052")
os.environ.setdefault("CLASSIFIER_GRPC", "localhost:50053")
os.environ.setdefault("SIMILARITY_GRPC", "localhost:50054")
os.environ.setdefault("TOXIC_DEVICE", "cpu")
os.environ.setdefault("LOG_LEVEL", "WARNING")


# ---------- Fixtures ----------------------------------------------------

@pytest.fixture
def fake_triton():
    """Возвращает FakeTritonAsyncClient — без GPU, без Triton."""
    from tests.helpers.fake_triton import FakeTritonAsyncClient
    return FakeTritonAsyncClient(url="localhost:8001")


@pytest.fixture
def nsfw_triton():
    """Fake Triton, который отвечает 'всё NSFW' (для проверки фильтрации)."""
    from tests.helpers.fake_triton import FakeTritonAsyncClient
    return FakeTritonAsyncClient(url="localhost:8001", force_nsfw=True)


@pytest.fixture
def sample_image_bytes() -> bytes:
    """Минимальный валидный RGB JPEG (1x1px, красный)."""
    from PIL import Image
    import io
    img = Image.new("RGB", (224, 224), color=(255, 50, 50))
    buf = io.BytesIO()
    img.save(buf, format="JPEG", quality=85)
    return buf.getvalue()


@pytest.fixture
def sample_image_url(sample_image_bytes, respx_mock) -> str:
    """Заглушка URL, возвращающая sample_image_bytes через respx."""
    import respx
    import httpx
    url = "https://fake-s3.example.com/test-image.jpg"
    respx_mock.get(url).mock(return_value=httpx.Response(200, content=sample_image_bytes))
    return url
