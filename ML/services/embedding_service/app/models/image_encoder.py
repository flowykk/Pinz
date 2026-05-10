from __future__ import annotations

from io import BytesIO

import open_clip
import torch
from PIL import Image

from app.config import settings


class ImageEncoder:
    """Wraps OpenAI CLIP ViT-L/14 for dense image embeddings."""

    def __init__(self) -> None:
        self._model, _, self._preprocess = open_clip.create_model_and_transforms(
            settings.image_embed_model,
            pretrained=settings.image_embed_pretrained,
            cache_dir=settings.model_cache_dir,
        )
        self._model = self._model.to(settings.device)
        self._model.eval()
        self._device = settings.device

    @torch.inference_mode()
    def encode(self, images_bytes: list[bytes]) -> list[list[float]]:
        tensors = []
        for raw in images_bytes:
            img = Image.open(BytesIO(raw)).convert("RGB")
            tensors.append(self._preprocess(img))
        batch = torch.stack(tensors).to(self._device)
        features = self._model.encode_image(batch)
        features = features / features.norm(dim=-1, keepdim=True)
        return features.cpu().tolist()
