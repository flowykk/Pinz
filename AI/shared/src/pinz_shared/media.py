"""Асинхронная загрузка медиа по presigned URL из payload MLTask."""

from __future__ import annotations

import asyncio
import io
from dataclasses import dataclass

import httpx
from PIL import Image

from .contracts import MediaItem
from .logging import get_logger

log = get_logger("media")


@dataclass(slots=True)
class FetchedMedia:
    media_id: str
    raw: bytes
    image: Image.Image | None  # None если decode упал или это не картинка


class MediaFetcher:
    """Качает по presigned URL'ам. Семафор ограничивает concurrency."""

    def __init__(self, *, concurrency: int = 16, timeout: int = 30) -> None:
        self._sem = asyncio.Semaphore(concurrency)
        self._client = httpx.AsyncClient(
            timeout=httpx.Timeout(timeout, connect=10),
            follow_redirects=True,
            limits=httpx.Limits(max_connections=concurrency * 2, max_keepalive_connections=concurrency),
        )

    async def close(self) -> None:
        await self._client.aclose()

    async def __aenter__(self) -> "MediaFetcher":
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:
        await self.close()

    async def fetch_one(self, item: MediaItem) -> FetchedMedia | None:
        async with self._sem:
            try:
                r = await self._client.get(item.get_url)
                r.raise_for_status()
            except Exception as e:  # noqa: BLE001
                log.warning("media fetch failed", media_id=item.media_id, error=str(e))
                return None

            raw = r.content
            img: Image.Image | None = None
            ct = (item.content_type or item.media_type or "").lower()
            if ct.startswith("image/") or not ct:
                try:
                    img = Image.open(io.BytesIO(raw)).convert("RGB")
                except Exception as e:  # noqa: BLE001
                    log.warning("image decode failed", media_id=item.media_id, error=str(e))
                    img = None
            return FetchedMedia(media_id=item.media_id, raw=raw, image=img)

    async def fetch_many(self, items: list[MediaItem]) -> dict[str, FetchedMedia]:
        results = await asyncio.gather(*(self.fetch_one(it) for it in items))
        return {r.media_id: r for r in results if r is not None}
