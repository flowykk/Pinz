from __future__ import annotations

import asyncio
import io

import grpc
import httpx
import numpy as np
from PIL import Image

from pinz_ai_proto import similarity_pb2, similarity_pb2_grpc
from pinz_shared.grpc_runtime import serve_grpc
from pinz_shared.logging import configure_logging, get_logger
from pinz_shared.phash import hamming, phash
from pinz_shared.union_find import UnionFind

from .settings import Settings


log = get_logger("similarity")


class SimilarityServicer(similarity_pb2_grpc.SimilarityServicer):
    """Алгоритм:
       1. По всем картинкам считаем pHash.
       2. Любая пара с hamming(pHash) <= phash_max → union, дополнительно
          подтверждаем cosine(SigLIP) >= cos_min (даёт устойчивость к шуму).
       3. По оставшимся (не объединённым по pHash) — берём cosine между всеми парами
          с участием new-медиа; если >= cos_min → union.
       4. Возвращаем группы размера >= 2.
    """

    def __init__(
        self,
        http: httpx.AsyncClient,
        sem: asyncio.Semaphore,
        phash_max: int,
        cos_min: float,
    ) -> None:
        self._http = http
        self._sem = sem
        self._phash_max = phash_max
        self._cos_min = cos_min

    async def _fetch_phash(self, url: str) -> int | None:
        async with self._sem:
            try:
                r = await self._http.get(url)
                r.raise_for_status()
                img = Image.open(io.BytesIO(r.content)).convert("RGB")
                return phash(img)
            except Exception as e:  # noqa: BLE001
                log.warning("phash fetch/decode failed", error=str(e))
                return None

    async def FindSimilarGroups(
        self,
        request: similarity_pb2.FindSimilarGroupsRequest,
        context: grpc.aio.ServicerContext,
    ) -> similarity_pb2.FindSimilarGroupsResponse:
        media = list(request.media)
        if len(media) < 2:
            return similarity_pb2.FindSimilarGroupsResponse()

        ids = [m.media_id for m in media]
        is_new = {m.media_id: m.is_new for m in media}
        vecs: dict[str, np.ndarray] = {}
        for m in media:
            if m.embedding and m.embedding.vector:
                v = np.asarray(m.embedding.vector, dtype=np.float32)
                n = np.linalg.norm(v) + 1e-12
                vecs[m.media_id] = v / n

        # 1. pHash для всех
        phashes = await asyncio.gather(*(self._fetch_phash(m.get_url) for m in media))
        ph_by_id: dict[str, int] = {
            mid: ph for mid, ph in zip(ids, phashes) if ph is not None
        }

        uf = UnionFind(ids)

        # 2/3. pairwise сравнение, фокус на парах с участием new
        for i in range(len(ids)):
            for j in range(i + 1, len(ids)):
                a, b = ids[i], ids[j]
                if not (is_new[a] or is_new[b]):
                    continue

                ham = None
                if a in ph_by_id and b in ph_by_id:
                    ham = hamming(ph_by_id[a], ph_by_id[b])

                cos = None
                if a in vecs and b in vecs:
                    cos = float(np.dot(vecs[a], vecs[b]))

                near_dup = ham is not None and ham <= self._phash_max
                semantic_dup = cos is not None and cos >= self._cos_min
                
                if (near_dup and (cos is None or cos >= 0.85)) or semantic_dup:
                    uf.union(a, b)

        groups = uf.groups(min_size=2)
        log.info(
            "similarity done",
            pin_id=request.pin_id,
            n_media=len(ids),
            n_groups=len(groups),
        )
        return similarity_pb2.FindSimilarGroupsResponse(
            groups=[similarity_pb2.Group(media_ids=g) for g in groups]
        )


async def main() -> None:
    settings = Settings()
    configure_logging(settings.log_level, settings.service_name)

    http = httpx.AsyncClient(
        timeout=httpx.Timeout(settings.http_fetch_timeout, connect=10),
        follow_redirects=True,
    )
    sem = asyncio.Semaphore(settings.http_fetch_concurrency)

    servicer = SimilarityServicer(
        http,
        sem,
        settings.similarity_phash_hamming_max,
        settings.similarity_cosine_min,
    )

    async def register(server: grpc.aio.Server) -> None:
        similarity_pb2_grpc.add_SimilarityServicer_to_server(servicer, server)

    try:
        await serve_grpc(settings.grpc_port, register, service_name=settings.service_name)
    finally:
        await http.aclose()


if __name__ == "__main__":
    asyncio.run(main())
