from __future__ import annotations

from qdrant_client import AsyncQdrantClient
from qdrant_client.models import Distance, PointStruct, VectorParams

from app.config import settings

_qdrant: AsyncQdrantClient | None = None


def get_qdrant() -> AsyncQdrantClient:
    global _qdrant
    if _qdrant is None:
        _qdrant = AsyncQdrantClient(url=settings.qdrant_url)
    return _qdrant


async def ensure_collection() -> None:
    qdrant = get_qdrant()
    collections = await qdrant.get_collections()
    names = [c.name for c in collections.collections]
    if settings.qdrant_collection_pins not in names:
        await qdrant.create_collection(
            collection_name=settings.qdrant_collection_pins,
            vectors_config={
                "text": VectorParams(size=settings.vector_dim_text, distance=Distance.COSINE),
                "image": VectorParams(size=settings.vector_dim_image, distance=Distance.COSINE),
            },
        )


async def upsert_pin(pin_id: str, text_vec: list[float], image_vec: list[float], payload: dict) -> None:
    qdrant = get_qdrant()
    await qdrant.upsert(
        collection_name=settings.qdrant_collection_pins,
        points=[
            PointStruct(
                id=pin_id,
                vector={"text": text_vec, "image": image_vec},
                payload=payload,
            )
        ],
    )


async def vector_search(
    query_vec: list[float],
    vector_name: str = "text",
    limit: int = 10,
) -> list[dict]:
    qdrant = get_qdrant()
    results = await qdrant.search(
        collection_name=settings.qdrant_collection_pins,
        query_vector=(vector_name, query_vec),
        limit=limit,
        with_payload=True,
    )
    return [{"pin_id": str(r.id), "score": r.score, **r.payload} for r in results]
