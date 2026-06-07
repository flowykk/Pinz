"""Прокси к Triton: SigLIP vision + text"""

from __future__ import annotations

import io

import numpy as np
from PIL import Image
from transformers import AutoTokenizer

from pinz_shared.triton import TritonAsyncClient


_MEAN = np.array([0.5, 0.5, 0.5], dtype=np.float32).reshape(1, 3, 1, 1)
_STD = np.array([0.5, 0.5, 0.5], dtype=np.float32).reshape(1, 3, 1, 1)


def preprocess_image_bytes(raw: bytes, size: int = 224) -> np.ndarray:
    img = Image.open(io.BytesIO(raw)).convert("RGB").resize((size, size), Image.Resampling.BILINEAR)
    arr = np.asarray(img, dtype=np.float32) / 255.0
    arr = arr.transpose(2, 0, 1)[None, ...]
    arr = (arr - _MEAN) / _STD
    return arr.astype(np.float32)


def _l2_normalize(x: np.ndarray) -> np.ndarray:
    n = np.linalg.norm(x, axis=-1, keepdims=True) + 1e-12
    return (x / n).astype(np.float32)


class SiglipImageEncoder:
    def __init__(self, triton: TritonAsyncClient, model_name: str) -> None:
        self._t = triton
        self._m = model_name

    async def encode(self, tensors: list[np.ndarray]) -> np.ndarray:
        if not tensors:
            return np.zeros((0,), dtype=np.float32)
        batch = np.concatenate(tensors, axis=0)
        out = await self._t.infer(self._m, {"pixel_values": batch}, ["image_embeds"])
        return _l2_normalize(out["image_embeds"])


class SiglipTextEncoder:
    def __init__(self, triton: TritonAsyncClient, model_name: str, hf_id: str) -> None:
        self._t = triton
        self._m = model_name
        self.tokenizer = AutoTokenizer.from_pretrained(hf_id)

    async def encode(self, texts: list[str]) -> np.ndarray:
        if not texts:
            return np.zeros((0,), dtype=np.float32)
        enc = self.tokenizer(
            texts,
            padding="max_length",
            truncation=True,
            max_length=64,
            return_tensors="np",
        )
        out = await self._t.infer(
            self._m,
            {"input_ids": enc["input_ids"].astype(np.int64)},
            ["text_embeds"],
        )
        return _l2_normalize(out["text_embeds"])
