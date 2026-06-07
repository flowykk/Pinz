"""gRPC сервер Moderation."""

from __future__ import annotations

import asyncio

import grpc
import httpx

from pinz_ai_proto import moderation_pb2, moderation_pb2_grpc
from pinz_shared.grpc_runtime import serve_grpc
from pinz_shared.logging import configure_logging, get_logger
from pinz_shared.triton import TritonAsyncClient

from .nsfw import NSFWClassifier, preprocess_bytes
from .settings import Settings
from .toxic import ToxicClassifier


log = get_logger("moderation")


class ModerationServicer(moderation_pb2_grpc.ModerationServicer):
    def __init__(
        self,
        nsfw: NSFWClassifier,
        toxic: ToxicClassifier,
        nsfw_threshold: float,
        http: httpx.AsyncClient,
        fetch_sem: asyncio.Semaphore,
    ) -> None:
        self._nsfw = nsfw
        self._toxic = toxic
        self._nsfw_th = nsfw_threshold
        self._http = http
        self._sem = fetch_sem

    async def _fetch_image(self, get_url: str):
        async with self._sem:
            try:
                r = await self._http.get(get_url)
                r.raise_for_status()
                return r.content
            except Exception as e:  # noqa: BLE001
                log.warning("media fetch failed", error=str(e))
                return None

    async def ModerateMedia(
        self,
        request: moderation_pb2.ModerateMediaRequest,
        context: grpc.aio.ServicerContext,
    ) -> moderation_pb2.ModerateMediaResponse:
        media_list = list(request.media)
        if not media_list:
            return moderation_pb2.ModerateMediaResponse()

        raws = await asyncio.gather(*(self._fetch_image(m.get_url) for m in media_list))

        tensors = []
        ids = []
        for m, raw in zip(media_list, raws):
            if raw is None:
                continue
            try:
                tensors.append(preprocess_bytes(raw))
                ids.append(m.media_id)
            except Exception as e:  # noqa: BLE001
                log.warning("preprocess failed", media_id=m.media_id, error=str(e))

        probs = await self._nsfw.predict_batch(tensors)
        nsfw_ids = [mid for mid, p in zip(ids, probs) if p >= self._nsfw_th]
        log.info(
            "moderate_media done",
            trip_id=request.trip_id,
            total=len(media_list),
            scored=len(ids),
            nsfw=len(nsfw_ids),
        )
        return moderation_pb2.ModerateMediaResponse(nsfw_media_ids=nsfw_ids)

    async def ModerateText(
        self,
        request: moderation_pb2.ModerateTextRequest,
        context: grpc.aio.ServicerContext,
    ) -> moderation_pb2.ModerateTextResponse:
        items = list(request.items)
        texts = [it.text or "" for it in items]
        probs = await self._toxic.predict(texts) if texts else []
        verdicts = []
        for it, p in zip(items, probs):
            verdicts.append(
                moderation_pb2.TextVerdict(
                    item_id=it.item_id,
                    entity_kind=it.entity_kind,
                    entity_id=it.entity_id,
                    field=it.field,
                    censored=self._toxic.is_toxic(p),
                )
            )
        log.info("moderate_text done", n=len(items))
        return moderation_pb2.ModerateTextResponse(verdicts=verdicts)


async def main() -> None:
    settings = Settings()
    configure_logging(settings.log_level, settings.service_name)

    triton = TritonAsyncClient(settings.triton_grpc_url)
    await triton.connect()
    nsfw = NSFWClassifier(triton=triton, model_name=settings.nsfw_model_name)
    toxic = ToxicClassifier(
        model_id=settings.toxic_model_id,
        device=settings.toxic_device,
        max_len=settings.toxic_max_len,
        threshold=settings.toxic_threshold,
    )

    http = httpx.AsyncClient(
        timeout=httpx.Timeout(settings.http_fetch_timeout, connect=10),
        follow_redirects=True,
    )
    sem = asyncio.Semaphore(settings.http_fetch_concurrency)

    servicer = ModerationServicer(nsfw, toxic, settings.nsfw_threshold, http, sem)

    async def register(server: grpc.aio.Server) -> None:
        moderation_pb2_grpc.add_ModerationServicer_to_server(servicer, server)

    try:
        await serve_grpc(
            settings.grpc_port,
            register,
            service_name=settings.service_name,
        )
    finally:
        await http.aclose()
        await triton.close()


if __name__ == "__main__":
    asyncio.run(main())
