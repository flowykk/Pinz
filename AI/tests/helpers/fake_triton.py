"""FakeTritonAsyncClient — подмена TritonAsyncClient для тестов без GPU.

Возвращает:
  - siglip_vision / siglip_text → случайные L2-нормированные векторы dim=768
  - nsfw_detector               → логиты [10, -10] (safe) или [-10, 10] (nsfw)
                                   управляется флагом force_nsfw=True

Использование в тестах:
    from tests.helpers.fake_triton import FakeTritonAsyncClient
    monkeypatch.setattr("pinz_shared.triton.TritonAsyncClient", FakeTritonAsyncClient)
"""

from __future__ import annotations

import numpy as np


class FakeTritonAsyncClient:
    """Drop-in замена TritonAsyncClient; не требует Triton / GPU."""

    EMBEDDING_DIM = 768

    def __init__(self, url: str, *, force_nsfw: bool = False, seed: int = 42) -> None:
        self._url = url
        self._force_nsfw = force_nsfw
        self._rng = np.random.default_rng(seed)

    async def connect(self) -> None:
        """Нет реального соединения — сразу готов."""

    async def close(self) -> None:
        pass

    async def infer(
        self,
        model: str,
        inputs: dict[str, np.ndarray],
        outputs: list[str],
        *,
        model_version: str = "",
    ) -> dict[str, np.ndarray]:
        if model == "nsfw_detector":
            return self._nsfw(inputs)
        if model in ("siglip_vision", "siglip_text"):
            return self._embedding(inputs, outputs)
        raise ValueError(f"FakeTriton: unknown model '{model}'")

    # ------------------------------------------------------------------
    def _nsfw(self, inputs: dict[str, np.ndarray]) -> dict[str, np.ndarray]:
        batch = list(inputs.values())[0]
        n = batch.shape[0]
        if self._force_nsfw:
            logits = np.tile([-10.0, 10.0], (n, 1)).astype(np.float32)
        else:
            logits = np.tile([10.0, -10.0], (n, 1)).astype(np.float32)
        return {"logits": logits}

    def _embedding(
        self, inputs: dict[str, np.ndarray], outputs: list[str]
    ) -> dict[str, np.ndarray]:
        batch = list(inputs.values())[0]
        n = batch.shape[0]
        raw = self._rng.standard_normal((n, self.EMBEDDING_DIM)).astype(np.float32)
        norms = np.linalg.norm(raw, axis=-1, keepdims=True) + 1e-12
        embs = raw / norms
        key = outputs[0]  # "image_embeds" or "text_embeds"
        return {key: embs}
