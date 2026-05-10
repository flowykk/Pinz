from __future__ import annotations

import torch
from sentence_transformers import SentenceTransformer

from app.config import settings


class TextEncoder:
    """Wraps multilingual-e5-large for dense text embeddings.

    E5 models require the query/passage prefix for best retrieval quality:
    - "query: <text>" for search queries
    - "passage: <text>" for documents to be indexed
    """

    def __init__(self) -> None:
        self._model = SentenceTransformer(
            settings.text_embed_model,
            cache_folder=settings.model_cache_dir,
            device=settings.device,
        )

    def encode(self, texts: list[str], is_query: bool = False) -> list[list[float]]:
        prefix = "query" if is_query else "passage"
        prefixed = [f"{prefix}: {t}" for t in texts]
        embeddings = self._model.encode(
            prefixed,
            normalize_embeddings=True,
            show_progress_bar=False,
        )
        return embeddings.tolist()
