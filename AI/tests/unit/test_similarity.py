"""Unit-тесты SimilarityServicer — HTTP и Triton мокируются."""

import io
import asyncio
from unittest.mock import AsyncMock, MagicMock

import httpx
import numpy as np
import pytest
from PIL import Image

from pinz_shared.union_find import UnionFind


def _make_pil(color=(100, 150, 200)):
    return Image.new("RGB", (64, 64), color=color)


def _jpeg_bytes(color=(100, 150, 200)):
    img = _make_pil(color)
    buf = io.BytesIO()
    img.save(buf, format="JPEG")
    return buf.getvalue()


def _gradient_jpeg(start: int = 0, end: int = 200) -> bytes:
    """Градиентное изображение"""
    arr = np.zeros((64, 64, 3), dtype=np.uint8)
    arr[:, :, 0] = np.linspace(start, end, 64, dtype=np.uint8)
    arr[:, :, 1] = np.linspace(end, start, 64, dtype=np.uint8)
    arr[:, :, 2] = 128
    img = Image.fromarray(arr)
    buf = io.BytesIO()
    img.save(buf, format="JPEG")
    return buf.getvalue()


# Нормированный вектор
def _vec(seed=0, dim=768):
    rng = np.random.default_rng(seed)
    v = rng.standard_normal(dim).astype(np.float32)
    return v / np.linalg.norm(v)


# ---------- pHash-based grouping ----------------------------------------

@pytest.mark.asyncio
async def test_identical_images_grouped():
    """Два одинаковых изображения (одинаковый контент) должны дать одинаковый pHash."""
    from similarity_service.src.server import SimilarityServicer

    raw = _gradient_jpeg()
    http = MagicMock()
    http.get = AsyncMock(return_value=MagicMock(
        status_code=200,
        content=raw,
        raise_for_status=MagicMock(),
    ))
    sem = asyncio.Semaphore(4)
    svc = SimilarityServicer(http, sem, phash_max=10, cos_min=0.99)

    # Проверяем логику через _fetch_phash — одинаковый контент → одинаковый hash
    h1 = await svc._fetch_phash("url1")
    h2 = await svc._fetch_phash("url2")
    assert h1 is not None
    assert h1 == h2


@pytest.mark.asyncio
async def test_different_images_different_hash():
    """Случайные текстуры с разными seed'ами → разные pHash."""
    from pinz_shared.phash import phash, hamming

    def noise_jpeg(seed: int) -> bytes:
        rng = np.random.default_rng(seed)
        arr = rng.integers(0, 256, (64, 64, 3), dtype=np.uint8)
        img = Image.fromarray(arr)
        buf = io.BytesIO()
        img.save(buf, format="JPEG")
        return buf.getvalue()

    img1 = Image.open(io.BytesIO(noise_jpeg(seed=1))).convert("RGB")
    img2 = Image.open(io.BytesIO(noise_jpeg(seed=999))).convert("RGB")

    h1 = phash(img1)
    h2 = phash(img2)
    assert hamming(h1, h2) > 10


# ---------- cosine similarity grouping ---------------------------------

@pytest.mark.asyncio
async def test_cosine_similar_grouped():
    """Два вектора с cosine=0.99 → должны быть объединены."""
    from pinz_shared.union_find import UnionFind

    v1 = _vec(seed=1)
    # v2 ≈ v1, tiny perturbation
    v2 = v1 + np.random.default_rng(99).standard_normal(768).astype(np.float32) * 0.01
    v2 /= np.linalg.norm(v2)

    cosine = float(np.dot(v1, v2))
    assert cosine >= 0.92, f"cosine too low: {cosine}"

    uf = UnionFind(["a", "b"])
    if cosine >= 0.92:
        uf.union("a", "b")
    groups = uf.groups(min_size=2)
    assert len(groups) == 1


@pytest.mark.asyncio
async def test_orthogonal_vectors_not_grouped():
    """Ортогональные векторы → не должны быть в одной группе."""
    rng = np.random.default_rng(7)
    v1 = rng.standard_normal(768).astype(np.float32)
    v1 /= np.linalg.norm(v1)
    # Случайный ортогональный
    v2 = rng.standard_normal(768).astype(np.float32)
    v2 -= v2.dot(v1) * v1
    v2 /= np.linalg.norm(v2) + 1e-12

    cosine = float(np.dot(v1, v2))
    assert abs(cosine) < 0.2  # ортогональные

    uf = UnionFind(["a", "b"])
    if cosine >= 0.92:
        uf.union("a", "b")
    assert uf.groups(min_size=2) == []
