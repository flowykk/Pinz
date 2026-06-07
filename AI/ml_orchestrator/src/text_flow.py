"""Обработка text_moderation flow"""

from __future__ import annotations

from pinz_shared import MLResultMessage, MLTextResult, MLTextTaskMessage, Flow

from .grpc_clients import GrpcClients, moderation_pb2


async def handle_text_task(task: MLTextTaskMessage, clients: GrpcClients) -> MLResultMessage:
    req = moderation_pb2.ModerateTextRequest(
        items=[
            moderation_pb2.TextRef(
                item_id=it.item_id,
                entity_kind=it.entity_kind,
                entity_id=it.entity_id,
                field=it.field,
                text=it.text,
            )
            for it in task.items
        ]
    )
    resp = await clients.moderation.ModerateText(req)

    return MLResultMessage(
        flow=Flow.TEXT_MODERATION,
        trip_id=task.trip_id,
        text_results=[
            MLTextResult(
                item_id=v.item_id,
                entity_kind=v.entity_kind,
                entity_id=v.entity_id,
                field=v.field,
                censored=v.censored,
            )
            for v in resp.verdicts
        ],
    )
