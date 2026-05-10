from __future__ import annotations

from io import BytesIO

import numpy as np
import opennsfw2 as n2
from PIL import Image

from app.config import settings


class ImageModerator:
    def __init__(self) -> None:
        # opennsfw2 loads weights lazily on first call
        self._threshold = settings.image_nsfw_threshold

    def predict(self, image_bytes: bytes) -> dict:
        image = Image.open(BytesIO(image_bytes)).convert("RGB")
        image_array = np.array(image)
        # predict_image returns (sfw_prob, nsfw_prob)
        _, nsfw_prob = n2.predict_image(image)
        nsfw_score = float(nsfw_prob)
        return {
            "nsfw": nsfw_score >= self._threshold,
            "score": nsfw_score,
        }
