"""Обёртка над Qdrant для двух коллекций: pin_images и pin_texts."""

from __future__ import annotations

import uuid

from qdrant_client import AsyncQdrantClient, models

from pinz_shared.logging import get_logger

log = get_logger("qdrant")


class QdrantStore:
    def __init__(self, url: str, dim: int, col_images: str, col_texts: str) -> None:
        self._client = AsyncQdrantClient(url=url, prefer_grpc=False)
        self._dim = dim
        self._col_images = col_images
        self._col_texts = col_texts

    async def ensure_collections(self) -> None:
        for col in (self._col_images, self._col_texts):
            exists = await self._client.collection_exists(col)
            if not exists:
                await self._client.create_collection(
                    collection_name=col,
                    vectors_config=models.VectorParams(
                        size=self._dim, distance=models.Distance.COSINE
                    ),
                )
                # фильтр по trip_id, pin_id, category — индексы
                for field, schema in (
                    ("trip_id", models.PayloadSchemaType.KEYWORD),
                    ("pin_id", models.PayloadSchemaType.KEYWORD),
                    ("category", models.PayloadSchemaType.KEYWORD),
                ):
                    try:
                        await self._client.create_payload_index(col, field, schema)
                    except Exception as e:  # noqa: BLE001
                        log.warning("create_payload_index failed", col=col, field=field, error=str(e))

    async def upsert_image_vectors(
        self,
        *,
        trip_id: str,
        pin_id: str,
        category: str,
        tags: list[str],
        media_ids: list[str],
        vectors: list[list[float]],
    ) -> int:
        if not vectors:
            return 0
        points = [
            models.PointStruct(
                id=str(uuid.uuid5(uuid.NAMESPACE_URL, f"img:{pin_id}:{mid}")),
                vector=vec,
                payload={
                    "trip_id": trip_id,
                    "pin_id": pin_id,
                    "media_id": mid,
                    "category": category,
                    "tags": tags,
                },
            )
            for mid, vec in zip(media_ids, vectors)
        ]
        await self._client.upsert(self._col_images, points=points, wait=True)
        return len(points)

    async def upsert_text_vector(
        self,
        *,
        trip_id: str,
        pin_id: str,
        category: str,
        tags: list[str],
        vector: list[float],
    ) -> int:
        point = models.PointStruct(
            id=str(uuid.uuid5(uuid.NAMESPACE_URL, f"txt:{pin_id}")),
            vector=vector,
            payload={
                "trip_id": trip_id,
                "pin_id": pin_id,
                "category": category,
                "tags": tags,
            },
        )
        await self._client.upsert(self._col_texts, points=[point], wait=True)
        return 1

    async def search_images(
        self, *, trip_id: str, query_vec: list[float], top_k: int = 20
    ) -> list[tuple[str, float, str]]:
        """Поиск media по семантике; группирует по pin_id (берём лучший score)."""
        hits = await self._client.search(
            self._col_images,
            query_vector=query_vec,
            limit=top_k * 3,
            query_filter=models.Filter(
                must=[models.FieldCondition(key="trip_id", match=models.MatchValue(value=trip_id))]
            ),
        )
        best: dict[str, tuple[float, str]] = {}
        for h in hits:
            pin = h.payload.get("pin_id")
            if pin is None:
                continue
            cur = best.get(pin)
            if cur is None or h.score > cur[0]:
                best[pin] = (float(h.score), h.payload.get("category", "custom"))
        return [(p, s, c) for p, (s, c) in best.items()][:top_k]

    async def search_text(
        self, *, trip_id: str, query_vec: list[float], top_k: int = 20
    ) -> list[tuple[str, float, str]]:
        hits = await self._client.search(
            self._col_texts,
            query_vector=query_vec,
            limit=top_k,
            query_filter=models.Filter(
                must=[models.FieldCondition(key="trip_id", match=models.MatchValue(value=trip_id))]
            ),
        )
        return [(h.payload["pin_id"], float(h.score), h.payload.get("category", "custom")) for h in hits]

    async def close(self) -> None:
        await self._client.close()
