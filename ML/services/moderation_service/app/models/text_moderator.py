from __future__ import annotations

import torch
from transformers import AutoTokenizer, AutoModelForSequenceClassification

from app.config import settings


class TextModerator:
    def __init__(self) -> None:
        self._tokenizer = AutoTokenizer.from_pretrained(
            settings.text_model_name,
            cache_dir=settings.model_cache_dir,
        )
        self._model = AutoModelForSequenceClassification.from_pretrained(
            settings.text_model_name,
            cache_dir=settings.model_cache_dir,
        ).to(settings.device)
        self._model.eval()
        self._threshold = settings.text_toxicity_threshold

    @torch.inference_mode()
    def predict(self, text: str) -> dict:
        inputs = self._tokenizer(
            text,
            return_tensors="pt",
            truncation=True,
            max_length=512,
        ).to(settings.device)
        logits = self._model(**inputs).logits
        probs = torch.softmax(logits, dim=-1).squeeze().tolist()

        # rubert-tiny-toxicity: labels [non-toxic, toxic]
        if isinstance(probs, float):
            probs = [1 - probs, probs]
        toxic_score = float(probs[-1])
        return {
            "toxic": toxic_score >= self._threshold,
            "score": toxic_score,
        }
