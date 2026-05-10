from __future__ import annotations

from io import BytesIO

import torch
import torch.nn as nn
from PIL import Image
from torchvision import models, transforms

from app.config import settings

_IMAGENET_MEAN = [0.485, 0.456, 0.406]
_IMAGENET_STD  = [0.229, 0.224, 0.225]

_transform = transforms.Compose([
    transforms.Resize(380),
    transforms.CenterCrop(380),
    transforms.ToTensor(),
    transforms.Normalize(mean=_IMAGENET_MEAN, std=_IMAGENET_STD),
])


class PinClassifier:
    def __init__(self) -> None:
        backbone = models.efficientnet_b4(weights=None)
        backbone.classifier[1] = nn.Linear(
            backbone.classifier[1].in_features,
            settings.num_classes,
        )
        state = torch.load(
            settings.model_weights_path,
            map_location=settings.device,
            weights_only=True,
        )
        backbone.load_state_dict(state)
        self._model = backbone.to(settings.device).eval()
        self._top_k = settings.top_k
        self._device = settings.device

    @torch.inference_mode()
    def predict(self, image_bytes: bytes) -> list[dict]:
        img = Image.open(BytesIO(image_bytes)).convert("RGB")
        tensor = _transform(img).unsqueeze(0).to(self._device)
        logits = self._model(tensor)
        probs = torch.softmax(logits, dim=-1).squeeze()
        top_probs, top_idxs = probs.topk(self._top_k)
        return [
            {"class_id": int(idx), "score": float(prob)}
            for idx, prob in zip(top_idxs, top_probs)
        ]
