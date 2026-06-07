"""Search Service: гибридный поиск пинов.

Параллельно:
  - SigLIP text → Qdrant (image + text коллекции)
  - Meilisearch BM25 по name/description/tags

Объединение через Reciprocal Rank Fusion.
"""

from __future__ import annotations

import asyncio

import grpc
from meilisearch_python_sdk import AsyncClient as MeiliClient
from qdrant_client import AsyncQdrantClient, models

from pinz_ai_proto import (
    embedding_pb2,
    embedding_pb2_grpc,
    search_pb2,
    search_pb2_grpc,
)
from pinz_shared.grpc_runtime import serve_grpc
from pinz_shared.logging import configure_logging, get_logger

from .settings import Settings


log = get_logger("search")


CHANNEL_OPTS = [
    ("grpc.max_send_message_length", 64 * 1024 * 1024),
    ("grpc.max_receive_message_length", 64 * 1024 * 1024),
]


def rrf(rankings: list[list[tuple[str, str]]], k: int) -> list[tuple[str, float, str]]:
    """rankings: список ранжированных списков [(pin_id, category), ...].
    Возвращает [(pin_id, fused_score, category), ...]."""
    scores: dict[str, float] = {}
    categories: dict[str, str] = {}
    for ranked in rankings:
        for rank, (pid, cat) in enumerate(ranked):
            scores[pid] = scores.get(pid, 0.0) + 1.0 / (k + rank + 1)
            if pid not in categories or cat != "custom":
                categories[pid] = cat
    fused = sorted(scores.items(), key=lambda kv: kv[1], reverse=True)
    return [(p, s, categories.get(p, "custom")) for p, s in fused]


class SearchServicer(search_pb2_grpc.SearchServicer):
    def __init__(
        self,
        settings: Settings,
        emb_stub: embedding_pb2_grpc.EmbeddingStub,
        qdrant: AsyncQdrantClient,
        meili: MeiliClient,
    ) -> None:
        self._s = settings
        self._emb = emb_stub
        self._qdrant = qdrant
        self._meili = meili

    async def _ensure_meili_index(self) -> None:
        try:
            await self._meili.get_index(self._s.meili_index)
        except Exception:
            await self._meili.create_index(self._s.meili_index, primary_key="pin_id")
            idx = self._meili.index(self._s.meili_index)
            await idx.update_filterable_attributes(["trip_id", "category", "tags"])
            await idx.update_searchable_attributes(["name", "description", "tags"])

    async def IndexPin(self, request, context):
        await self._ensure_meili_index()
        idx = self._meili.index(self._s.meili_index)
        await idx.add_documents([
            {
                "pin_id": request.pin_id,
                "trip_id": request.trip_id,
                "name": request.name or "",
                "description": request.description or "",
                "category": request.category or "custom",
                "tags": list(request.tags),
            }
        ])
        return search_pb2.IndexPinResponse(ok=True)

    async def _qdrant_search(
        self, collection: str, trip_id: str, vec: list[float], top_k: int
    ) -> list[tuple[str, str]]:
        if not vec:
            return []
        try:
            hits = await self._qdrant.search(
                collection,
                query_vector=vec,
                limit=top_k * 3,
                query_filter=models.Filter(
                    must=[
                        models.FieldCondition(
                            key="trip_id", match=models.MatchValue(value=trip_id)
                        )
                    ]
                ),
            )
        except Exception as e:
            log.warning("qdrant search failed", col=collection, error=str(e))
            return []

        best: dict[str, tuple[float, str]] = {}
        for h in hits:
            pid = h.payload.get("pin_id") if h.payload else None
            if not pid:
                continue
            cat = (h.payload or {}).get("category", "custom")
            if pid not in best or h.score > best[pid][0]:
                best[pid] = (float(h.score), cat)
        ranked = sorted(best.items(), key=lambda kv: kv[1][0], reverse=True)
        return [(pid, cat) for pid, (_, cat) in ranked[:top_k]]

    async def _meili_search(self, trip_id: str, query: str, top_k: int) -> list[tuple[str, str]]:
        idx = self._meili.index(self._s.meili_index)
        try:
            res = await idx.search(
                query,
                limit=top_k,
                filter=f'trip_id = "{trip_id}"',
                attributes_to_retrieve=["pin_id", "category"],
            )
        except Exception as e:
            log.warning("meili search failed", error=str(e))
            return []
        return [(h["pin_id"], h.get("category", "custom")) for h in res.hits]

    async def SearchPins(self, request, context):
        top_k = request.top_k or self._s.default_top_k

        # SigLIP text-эмбеддинг для query
        emb_resp = await self._emb.EmbedTexts(
            embedding_pb2.EmbedTextsRequest(ids=[""], texts=[request.query])
        )
        vec: list[float] = []
        if emb_resp.embeddings:
            vec = list(emb_resp.embeddings[0].vector)

        img_ranked, txt_ranked, bm25 = await asyncio.gather(
            self._qdrant_search(self._s.qdrant_collection_images, request.trip_id, vec, top_k),
            self._qdrant_search(self._s.qdrant_collection_text, request.trip_id, vec, top_k),
            self._meili_search(request.trip_id, request.query, top_k),
        )

        fused = rrf([img_ranked, txt_ranked, bm25], k=self._s.rrf_k)[:top_k]
        return search_pb2.SearchPinsResponse(
            hits=[
                search_pb2.SearchHit(pin_id=p, score=s, category=c)
                for p, s, c in fused
            ]
        )


async def main() -> None:
    settings = Settings()
    configure_logging(settings.log_level, settings.service_name)

    qdrant = AsyncQdrantClient(url=settings.qdrant_url, prefer_grpc=False)
    meili = MeiliClient(url=settings.meili_url, api_key=settings.meili_master_key)
    chan = grpc.aio.insecure_channel(settings.embedding_grpc, options=CHANNEL_OPTS)
    emb_stub = embedding_pb2_grpc.EmbeddingStub(chan)

    servicer = SearchServicer(settings, emb_stub, qdrant, meili)
    await servicer._ensure_meili_index()

    async def register(server: grpc.aio.Server) -> None:
        search_pb2_grpc.add_SearchServicer_to_server(servicer, server)

    try:
        await serve_grpc(settings.grpc_port, register, service_name=settings.service_name)
    finally:
        await chan.close()
        await qdrant.close()
        await meili.aclose()


if __name__ == "__main__":
    asyncio.run(main())
