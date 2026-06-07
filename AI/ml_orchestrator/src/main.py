"""Точка входа ML Orchestrator. Поднимает NATS pull-consumer и роутит flow."""

from __future__ import annotations

import asyncio
import json
import signal

from nats.aio.msg import Msg
from pinz_shared import MLTaskMessage, MLTextTaskMessage, Flow
from pinz_shared.logging import configure_logging, get_logger
from pinz_shared.nats_client import (
    ensure_pull_subscription,
    ensure_streams_local_dev,
    nats_connect,
    publish_result,
    run_pull_loop,
)

from .grpc_clients import GrpcClients
from .media_flow import handle_media_task
from .settings import Settings
from .text_flow import handle_text_task


async def main() -> None:
    settings = Settings()
    configure_logging(settings.log_level, settings.service_name)
    log = get_logger("orchestrator")

    log.info(
        "starting",
        nats_url=settings.nats_url,
        bootstrap_local=settings.bootstrap_streams_local,
    )

    clients = GrpcClients(
        moderation=settings.moderation_grpc,
        embedding=settings.embedding_grpc,
        classifier=settings.classifier_grpc,
        similarity=settings.similarity_grpc,
    )

    async with nats_connect(settings.nats_url, settings.nats_token) as nc:
        js = nc.jetstream()

        if settings.bootstrap_streams_local:
            await ensure_streams_local_dev(js)

        sub = await ensure_pull_subscription(js)
        log.info("subscribed", durable="ml-workers", subject="ml.tasks.>")

        stop = asyncio.Event()
        loop = asyncio.get_running_loop()
        for s in (signal.SIGINT, signal.SIGTERM):
            try:
                loop.add_signal_handler(s, stop.set)
            except NotImplementedError:
                pass

        async def handler(data: bytes, msg: Msg) -> None:
            try:
                payload = json.loads(data)
            except json.JSONDecodeError:
                log.warning("invalid json — ack and drop", subject=msg.subject)
                await msg.ack()
                return

            flow_str = payload.get("flow")
            try:
                flow = Flow(flow_str)
            except ValueError:
                log.warning("unknown flow — ack and drop", flow=flow_str)
                await msg.ack()
                return

            log.info("task received", flow=flow.value, subject=msg.subject)

            try:
                if flow == Flow.TEXT_MODERATION:
                    task_t = MLTextTaskMessage.model_validate(payload)
                    result = await handle_text_task(task_t, clients)
                else:
                    task_m = MLTaskMessage.model_validate(payload)
                    result = await handle_media_task(task_m, clients)
            except Exception as e:
                log.error("flow handler failed — NAK", flow=flow.value, error=str(e))
                await msg.nak(delay=10)
                return

            try:
                await publish_result(js, flow.value, result.to_json_bytes())
                await msg.ack()
                log.info("task completed", flow=flow.value)
            except Exception as e:
                log.error("publish result failed — NAK", error=str(e))
                await msg.nak(delay=10)

        try:
            await run_pull_loop(
                js,
                sub,
                handler,
                batch=settings.pull_batch,
                fetch_timeout=settings.pull_timeout,
                stop_event=stop,
            )
        finally:
            await clients.close()


if __name__ == "__main__":
    asyncio.run(main())
