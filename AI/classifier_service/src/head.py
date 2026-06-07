from __future__ import annotations

import os
from pathlib import Path

import numpy as np
import torch
import torch.nn as nn


class PinHead(nn.Module):
    def __init__(self, emb_dim: int, n_classes: int) -> None:
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(emb_dim, 256),
            nn.GELU(),
            nn.Dropout(0.1),
            nn.Linear(256, n_classes),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:  # x: (B, emb_dim)
        return self.net(x)


def _aggregate(embs: list[list[float]]) -> np.ndarray:
    if not embs:
        return np.zeros((0,), dtype=np.float32)
    a = np.asarray(embs, dtype=np.float32)
    mean = a.mean(axis=0)
    n = np.linalg.norm(mean) + 1e-12
    return (mean / n).astype(np.float32)


class Classifier:
    def __init__(
        self,
        emb_dim: int,
        categories: list[str],
        head_path: str,
        prototypes_path: str,
    ) -> None:
        self.categories = categories
        self.emb_dim = emb_dim
        self.head: PinHead | None = None
        self.prototypes: np.ndarray | None = None

        if os.path.exists(head_path):
            self.head = PinHead(emb_dim, len(categories))
            self.head.load_state_dict(torch.load(head_path, map_location="cpu"))
            self.head.eval()
        elif os.path.exists(prototypes_path):
            self.prototypes = np.load(prototypes_path).astype(np.float32)
            assert self.prototypes.shape == (len(categories), emb_dim), (
                f"prototypes shape mismatch: {self.prototypes.shape}"
            )
        else:
            pass

    @torch.inference_mode()
    def predict(self, image_embeddings: list[list[float]]) -> tuple[str, float]:
        agg = _aggregate(image_embeddings)
        if agg.size == 0:
            return "custom", 0.0

        if self.head is not None:
            x = torch.from_numpy(agg).unsqueeze(0)
            logits = self.head(x)[0]
            probs = torch.softmax(logits, dim=-1).numpy()
            idx = int(np.argmax(probs))
            return self.categories[idx], float(probs[idx])

        if self.prototypes is not None:
            sims = self.prototypes @ agg
            # Softmax over cosine similarities → probability → confidence ∈ [0, 1]
            e = np.exp(sims - sims.max())
            probs = e / (e.sum() + 1e-12)
            idx = int(np.argmax(probs))
            return self.categories[idx], float(probs[idx])

        return "custom", 0.0
