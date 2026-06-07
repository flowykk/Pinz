"""Обработка media-flow: creation / add_media / pin_upload.creation / pin_upload.addition.

Шаги:
  1. Параллельный NSFW по new-медиа.
  2. Параллельные SigLIP-эмбеддинги по всем медиа пина (new+existing); для new — upsert в Qdrant.
  3. Similarity по всему набору медиа пина (только new↔* пары).
  4. Классификация — для new-пинов и для pin_upload.creation.
"""

from __future__ import annotations

import asyncio

from pinz_shared import (
    Category,
    Flow,
    MLResultMessage,
    MLTaskMessage,
    PinSuggestion,
    PinItem,
)
from pinz_shared.logging import get_logger

from .grpc_clients import (
    GrpcClients,
    classifier_pb2,
    common_pb2,
    embedding_pb2,
    moderation_pb2,
    similarity_pb2,
)

log = get_logger("media_flow")


def _to_media_ref(m) -> common_pb2.MediaRef:
    return common_pb2.MediaRef(
        media_id=m.media_id,
        get_url=m.get_url,
        content_type=m.content_type or m.media_type or "",
        is_new=m.is_new,
    )


def _allow_suggestions(flow: Flow, pin: PinItem) -> bool:
    if flow == Flow.PIN_UPLOAD_ADDITION:
        return False
    if flow == Flow.PIN_UPLOAD_CREATION:
        return True
    return bool(pin.is_new)


async def handle_media_task(task: MLTaskMessage, clients: GrpcClients) -> MLResultMessage:
    nsfw_ids: list[str] = []
    similar_groups: list[list[str]] = []
    pin_suggestions: list[PinSuggestion] = []

    # ---------- 1) NSFW по new-медиа всего payload (батч) ----------
    new_media_refs = [
        _to_media_ref(m)
        for pin in task.pins
        for m in pin.media
        if m.is_new
    ]
    if new_media_refs:
        mod_resp = await clients.moderation.ModerateMedia(
            moderation_pb2.ModerateMediaRequest(trip_id=task.trip_id, media=new_media_refs)
        )
        nsfw_ids = list(mod_resp.nsfw_media_ids)

    # NSFW-медиа исключаем из дальнейших шагов (не эмбеддим, не классифицируем).
    nsfw_set = set(nsfw_ids)

    # ---------- 2/3/4) Per-pin pipeline параллельно ----------
    async def _process_pin(pin: PinItem) -> tuple[list[list[str]], PinSuggestion | None]:
        media_clean = [m for m in pin.media if m.media_id not in nsfw_set]
        if not media_clean:
            return [], None

        media_refs = [_to_media_ref(m) for m in media_clean]

        # ---- 2) эмбеддинги SigLIP ----
        emb_resp = await clients.embedding.EmbedImages(
            embedding_pb2.EmbedImagesRequest(
                media=media_refs,
                upsert_to_qdrant=False,  # сделаем upsert после классификации, с категорией
                trip_id=task.trip_id,
                pin_id=pin.pin_id,
            )
        )
        emb_by_id = {e.media_id: e for e in emb_resp.embeddings}

        # ---- 3) similarity ----
        sim_resp = await clients.similarity.FindSimilarGroups(
            similarity_pb2.FindSimilarGroupsRequest(
                pin_id=pin.pin_id,
                media=[
                    similarity_pb2.MediaWithEmbedding(
                        media_id=m.media_id,
                        is_new=m.is_new,
                        get_url=m.get_url,
                        embedding=emb_by_id.get(m.media_id),
                    )
                    for m in media_clean
                    if m.media_id in emb_by_id
                ],
            )
        )
        my_groups = [list(g.media_ids) for g in sim_resp.groups]

        # ---- 4) классификация (если нужно) ----
        suggestion: PinSuggestion | None = None
        if _allow_suggestions(task.flow, pin):
            cls_resp = await clients.classifier.ClassifyPin(
                classifier_pb2.ClassifyPinRequest(
                    pin_id=pin.pin_id,
                    image_embeddings=list(emb_by_id.values()),
                    description=pin.description or "",
                )
            )
            try:
                cat = Category(cls_resp.category)
            except ValueError:
                cat = Category.CUSTOM
            tags = [t[:15] for t in list(cls_resp.tags)[:10]]
            suggestion = PinSuggestion(pin_id=pin.pin_id, category=cat, tags=tags)

            # Upsert в Qdrant с категорией
            await clients.embedding.UpsertPinVectors(
                embedding_pb2.UpsertPinVectorsRequest(
                    trip_id=task.trip_id,
                    pin_id=pin.pin_id,
                    category=cat.value,
                    tags=tags,
                    image_embeddings=list(emb_by_id.values()),
                )
            )
        else:
            # Existing-пин — просто переиндексируем картинки с категорией "custom"
            await clients.embedding.UpsertPinVectors(
                embedding_pb2.UpsertPinVectorsRequest(
                    trip_id=task.trip_id,
                    pin_id=pin.pin_id,
                    category="custom",
                    image_embeddings=list(emb_by_id.values()),
                )
            )

        return my_groups, suggestion

    if task.pins:
        per_pin = await asyncio.gather(*(_process_pin(p) for p in task.pins))
        for groups, sug in per_pin:
            similar_groups.extend(groups)
            if sug is not None:
                pin_suggestions.append(sug)

    return MLResultMessage(
        flow=task.flow,
        trip_id=task.trip_id,
        session_id=task.session_id,
        similar_groups=similar_groups,
        nsfw_ids=nsfw_ids,
        pin_suggestions=pin_suggestions,
    )
