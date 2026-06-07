from __future__ import annotations

import asyncio

import grpc
import httpx
import numpy as np

from pinz_ai_proto import common_pb2, embedding_pb2, embedding_pb2_grpc
from pinz_shared.grpc_runtime import serve_grpc
from pinz_shared.logging import configure_logging, get_logger
from pinz_shared.triton import TritonAsyncClient

from .qdrant_store import QdrantStore
from .settings import Settings
from .siglip import SiglipImageEncoder, SiglipTextEncoder, preprocess_image_bytes


log = get_logger("embedding")


class EmbeddingServicer(embedding_pb2_grpc.EmbeddingServicer):
    def __init__(
        self,
        img: SiglipImageEncoder,
        txt: SiglipTextEncoder,
        qdrant: QdrantStore,
        http: httpx.AsyncClient,
        sem: asyncio.Semaphore,
    ) -> None:
        self._img = img
        self._txt = txt
        self._q = qdrant
        self._http = http
        self._sem = sem

    async def _fetch(self, url: str) -> bytes | None:
        async with self._sem:
            try:
                r = await self._http.get(url)
                r.raise_for_status()
                return r.content
            except Exception as e:  # noqa: BLE001
                log.warning("fetch failed", error=str(e))
                return None

    async def EmbedImages(self, request, context):
        media = list(request.media)
        if not media:
            return embedding_pb2.EmbedImagesResponse()

        raws = await asyncio.gather(*(self._fetch(m.get_url) for m in media))
        tensors, ids = [], []
        for m, raw in zip(media, raws):
            if raw is None:
                continue
            try:
                tensors.append(preprocess_image_bytes(raw))
                ids.append(m.media_id)
            except Exception as e:  # noqa: BLE001
                log.warning("preprocess failed", media_id=m.media_id, error=str(e))

        if not tensors:
            return embedding_pb2.EmbedImagesResponse()

        vecs = await self._img.encode(tensors)
        embeddings = [
            common_pb2.Embedding(media_id=mid, vector=vec.tolist())
            for mid, vec in zip(ids, vecs)
        ]

        if request.upsert_to_qdrant and request.pin_id:
            await self._q.upsert_image_vectors(
                trip_id=request.trip_id,
                pin_id=request.pin_id,
                category="custom",
                tags=[],
                media_ids=ids,
                vectors=[v.tolist() for v in vecs],
            )
        log.info("embed_images", n=len(embeddings))
        return embedding_pb2.EmbedImagesResponse(embeddings=embeddings)

    async def EmbedTexts(self, request, context):
        texts = list(request.texts)
        ids = list(request.ids) if request.ids else [""] * len(texts)
        if not texts:
            return embedding_pb2.EmbedTextsResponse()
        vecs = await self._txt.encode(texts)
        out = [
            common_pb2.Embedding(media_id=i, vector=v.tolist())
            for i, v in zip(ids, vecs)
        ]
        return embedding_pb2.EmbedTextsResponse(embeddings=out)

    async def UpsertPinVectors(self, request, context):
        n = 0
        if request.image_embeddings:
            n += await self._q.upsert_image_vectors(
                trip_id=request.trip_id,
                pin_id=request.pin_id,
                category=request.category or "custom",
                tags=list(request.tags),
                media_ids=[e.media_id for e in request.image_embeddings],
                vectors=[list(e.vector) for e in request.image_embeddings],
            )
        if request.text_embedding and list(request.text_embedding.vector):
            n += await self._q.upsert_text_vector(
                trip_id=request.trip_id,
                pin_id=request.pin_id,
                category=request.category or "custom",
                tags=list(request.tags),
                vector=list(request.text_embedding.vector),
            )
        log.info("upsert_pin_vectors", pin_id=request.pin_id, count=n)
        return embedding_pb2.UpsertPinVectorsResponse(upserted=n)


async def main() -> None:
    #configure_hf_ssl()
    settings = Settings()
    configure_logging(settings.log_level, settings.service_name)

    triton = TritonAsyncClient(settings.triton_grpc_url)
    await triton.connect()

    img_enc = SiglipImageEncoder(triton, settings.siglip_image_model)
    txt_enc = SiglipTextEncoder(triton, settings.siglip_text_model, settings.siglip_model_id)

    qdrant = QdrantStore(
        settings.qdrant_url,
        settings.embedding_dim,
        settings.qdrant_collection_images,
        settings.qdrant_collection_text,
    )
    await qdrant.ensure_collections()

    http = httpx.AsyncClient(
        timeout=httpx.Timeout(settings.http_fetch_timeout, connect=10),
        follow_redirects=True,
    )
    sem = asyncio.Semaphore(settings.http_fetch_concurrency)

    servicer = EmbeddingServicer(img_enc, txt_enc, qdrant, http, sem)

    async def register(server: grpc.aio.Server) -> None:
        embedding_pb2_grpc.add_EmbeddingServicer_to_server(servicer, server)

    try:
        await serve_grpc(settings.grpc_port, register, service_name=settings.service_name)
    finally:
        await http.aclose()
        await qdrant.close()
        await triton.close()


if __name__ == "__main__":
    asyncio.run(main())
