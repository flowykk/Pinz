from __future__ import annotations

import httpx
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from app.clients.es_client import bm25_search
from app.clients.qdrant_client import vector_search
from app.config import settings

router = APIRouter(prefix="/search", tags=["search"])


class SearchRequest(BaseModel):
    query: str
    limit: int = 10
    mode: str = "hybrid"  # "bm25" | "semantic" | "hybrid"


class PinResult(BaseModel):
    pin_id: str
    score: float
    title: str | None = None
    description: str | None = None


class SearchResponse(BaseModel):
    results: list[PinResult]
    mode: str


async def _get_query_embedding(text: str) -> list[float]:
    async with httpx.AsyncClient(base_url=settings.embedding_service_url) as client:
        resp = await client.post("/embed/text", json={"texts": [text], "is_query": True})
        resp.raise_for_status()
    return resp.json()["embeddings"][0]


def _reciprocal_rank_fusion(
    bm25_hits: list[dict],
    vector_hits: list[dict],
    k: int = 60,
) -> list[dict]:
    """Merge BM25 and vector rankings with RRF."""
    scores: dict[str, float] = {}
    meta: dict[str, dict] = {}

    for rank, hit in enumerate(bm25_hits):
        pid = hit["pin_id"]
        scores[pid] = scores.get(pid, 0.0) + 1.0 / (k + rank + 1)
        meta[pid] = hit

    for rank, hit in enumerate(vector_hits):
        pid = hit["pin_id"]
        scores[pid] = scores.get(pid, 0.0) + 1.0 / (k + rank + 1)
        meta.setdefault(pid, hit)

    merged = sorted(scores.items(), key=lambda x: x[1], reverse=True)
    return [{"pin_id": pid, "score": sc, **meta[pid]} for pid, sc in merged]


@router.post("/pins", response_model=SearchResponse)
async def search_pins(body: SearchRequest) -> SearchResponse:
    if not body.query.strip():
        raise HTTPException(status_code=422, detail="query must not be empty")

    limit = min(body.limit, 50)

    if body.mode == "bm25":
        hits = await bm25_search(body.query, size=limit)

    elif body.mode == "semantic":
        vec = await _get_query_embedding(body.query)
        hits = await vector_search(vec, vector_name="text", limit=limit)

    else:  # hybrid (default)
        bm25_hits = await bm25_search(body.query, size=settings.rrf_window)
        vec = await _get_query_embedding(body.query)
        vec_hits = await vector_search(vec, vector_name="text", limit=settings.rrf_window)
        hits = _reciprocal_rank_fusion(bm25_hits, vec_hits, k=settings.rrf_k)[:limit]

    results = [
        PinResult(
            pin_id=h["pin_id"],
            score=h["score"],
            title=h.get("title"),
            description=h.get("description"),
        )
        for h in hits
    ]
    return SearchResponse(results=results, mode=body.mode)
