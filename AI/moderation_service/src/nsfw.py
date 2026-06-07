"""NSFW-классификатор через Triton (Falconsai/nsfw_image_detection, экспорт ONNX)."""

from __future__ import annotations

import io
from dataclasses import dataclass

import numpy as np
from PIL import Image

from pinz_shared.triton import TritonAsyncClient


_MEAN = np.array([0.5, 0.5, 0.5], dtype=np.float32).reshape(1, 3, 1, 1)
_STD = np.array([0.5, 0.5, 0.5], dtype=np.float32).reshape(1, 3, 1, 1)


def _preprocess(img: Image.Image, size: int = 224) -> np.ndarray:
    img = img.convert("RGB").resize((size, size), Image.Resampling.BILINEAR)
    arr = np.asarray(img, dtype=np.float32) / 255.0  
    arr = arr.transpose(2, 0, 1)[None, ...]      
    arr = (arr - _MEAN) / _STD
    return arr.astype(np.float32)


def preprocess_bytes(raw: bytes, size: int = 224) -> np.ndarray:
    img = Image.open(io.BytesIO(raw)).convert("RGB")
    return _preprocess(img, size)


@dataclass(slots=True)
class NSFWClassifier:
    triton: TritonAsyncClient
    model_name: str

    async def predict_batch(self, tensors: list[np.ndarray]) -> list[float]:
        """Возвращает вероятность NSFW (класс 1) на каждое изображение."""
        if not tensors:
            return []
        batch = np.concatenate(tensors, axis=0)  # N 3 224 224
        out = await self.triton.infer(
            self.model_name,
            inputs={"pixel_values": batch},
            outputs=["logits"],
        )
        logits = out["logits"]  # N x 2q
        e = np.exp(logits - logits.max(axis=1, keepdims=True))
        probs = e / e.sum(axis=1, keepdims=True)
        return probs[:, 1].astype(float).tolist()
