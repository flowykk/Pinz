"""Unit-тесты NSFWClassifier — Triton подменён FakeTritonAsyncClient."""

import io
import numpy as np
import pytest
from PIL import Image

from tests.helpers.fake_triton import FakeTritonAsyncClient
from moderation_service.src.nsfw import NSFWClassifier, preprocess_bytes


def _make_jpeg(color=(255, 100, 50), size=224) -> bytes:
    img = Image.new("RGB", (size, size), color=color)
    buf = io.BytesIO()
    img.save(buf, format="JPEG", quality=85)
    return buf.getvalue()


# ---------- preprocess --------------------------------------------------

def test_preprocess_shape():
    raw = _make_jpeg()
    tensor = preprocess_bytes(raw)
    assert tensor.shape == (1, 3, 224, 224)
    assert tensor.dtype == np.float32


def test_preprocess_normalized():
    """Значения должны быть в разумном диапазоне после нормализации."""
    raw = _make_jpeg()
    tensor = preprocess_bytes(raw)
    # mean=0.5, std=0.5 → pixel=1.0 → (1-0.5)/0.5 = 1.0 → ok
    assert tensor.min() >= -3.0
    assert tensor.max() <= 3.0


def test_preprocess_different_sizes():
    """Должен работать с любым размером входного изображения."""
    for size in (64, 128, 512):
        raw = _make_jpeg(size=size)
        t = preprocess_bytes(raw, size=224)
        assert t.shape == (1, 3, 224, 224)


# ---------- NSFWClassifier ----------------------------------------------

@pytest.mark.asyncio
async def test_predict_batch_safe():
    triton = FakeTritonAsyncClient(url="fake", force_nsfw=False)
    clf = NSFWClassifier(triton=triton, model_name="nsfw_detector")
    tensors = [preprocess_bytes(_make_jpeg()) for _ in range(3)]
    probs = await clf.predict_batch(tensors)
    assert len(probs) == 3
    for p in probs:
        # FakeTriton (safe) → logits [10, -10] → softmax NSFW ≈ 0
        assert p < 0.01


@pytest.mark.asyncio
async def test_predict_batch_nsfw():
    triton = FakeTritonAsyncClient(url="fake", force_nsfw=True)
    clf = NSFWClassifier(triton=triton, model_name="nsfw_detector")
    tensors = [preprocess_bytes(_make_jpeg())]
    probs = await clf.predict_batch(tensors)
    assert len(probs) == 1
    # force_nsfw → logits [-10, 10] → softmax NSFW ≈ 1
    assert probs[0] > 0.99


@pytest.mark.asyncio
async def test_predict_batch_empty():
    triton = FakeTritonAsyncClient(url="fake")
    clf = NSFWClassifier(triton=triton, model_name="nsfw_detector")
    probs = await clf.predict_batch([])
    assert probs == []


@pytest.mark.asyncio
async def test_predict_batch_single():
    triton = FakeTritonAsyncClient(url="fake", force_nsfw=False)
    clf = NSFWClassifier(triton=triton, model_name="nsfw_detector")
    tensors = [preprocess_bytes(_make_jpeg())]
    probs = await clf.predict_batch(tensors)
    assert len(probs) == 1
    assert 0.0 <= probs[0] <= 1.0
