"""Текстовая модерация"""

from __future__ import annotations

import asyncio
from dataclasses import dataclass

import torch
from transformers import AutoModelForSequenceClassification, AutoTokenizer


@dataclass
class ToxicClassifier:
    model_id: str
    device: str
    max_len: int
    threshold: float

    def __post_init__(self) -> None:
        self.tokenizer = AutoTokenizer.from_pretrained(self.model_id)
        self.model = AutoModelForSequenceClassification.from_pretrained(self.model_id)
        self.model.to(self.device).eval()
        self._toxic_id = 1
        for i, lbl in self.model.config.id2label.items():
            if str(lbl).lower().startswith("toxic"):
                self._toxic_id = int(i)
                break

    @torch.inference_mode()
    def _predict_sync(self, texts: list[str]) -> list[float]:
        if not texts:
            return []
        enc = self.tokenizer(
            texts,
            padding=True,
            truncation=True,
            max_length=self.max_len,
            return_tensors="pt",
        ).to(self.device)
        out = self.model(**enc).logits
        probs = torch.softmax(out, dim=-1)[:, self._toxic_id].cpu().tolist()
        return probs

    async def predict(self, texts: list[str]) -> list[float]:
        return await asyncio.to_thread(self._predict_sync, texts)

    def is_toxic(self, p: float) -> bool:
        return p >= self.threshold
