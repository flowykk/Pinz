from __future__ import annotations

from elasticsearch import AsyncElasticsearch

from app.config import settings

_es: AsyncElasticsearch | None = None

PIN_MAPPING = {
    "mappings": {
        "properties": {
            "pin_id":      {"type": "keyword"},
            "user_id":     {"type": "keyword"},
            "title":       {"type": "text", "analyzer": "standard"},
            "description": {"type": "text", "analyzer": "standard"},
            "location":    {"type": "text"},
            "tags":        {"type": "keyword"},
            "created_at":  {"type": "date"},
        }
    }
}


def get_es() -> AsyncElasticsearch:
    global _es
    if _es is None:
        _es = AsyncElasticsearch(settings.elasticsearch_url)
    return _es


async def ensure_index() -> None:
    es = get_es()
    exists = await es.indices.exists(index=settings.elasticsearch_index_pins)
    if not exists:
        await es.indices.create(
            index=settings.elasticsearch_index_pins,
            body=PIN_MAPPING,
        )


async def index_pin(pin: dict) -> None:
    es = get_es()
    await es.index(
        index=settings.elasticsearch_index_pins,
        id=pin["pin_id"],
        document=pin,
    )


async def bm25_search(query: str, size: int = 10) -> list[dict]:
    es = get_es()
    resp = await es.search(
        index=settings.elasticsearch_index_pins,
        body={
            "size": size,
            "query": {
                "multi_match": {
                    "query": query,
                    "fields": ["title^2", "description", "location", "tags"],
                }
            },
        },
    )
    return [
        {"pin_id": h["_id"], "score": h["_score"], **h["_source"]}
        for h in resp["hits"]["hits"]
    ]
