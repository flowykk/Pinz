"""Тонкая обёртка над nats-py для durable pull-consumer'а 'ml-workers'."""

from __future__ import annotations

import asyncio
from collections.abc import AsyncIterator, Callable, Awaitable
from contextlib import asynccontextmanager
from typing import Any

import nats
from nats.aio.client import Client as NATS
from nats.aio.msg import Msg
from nats.errors import TimeoutError as NATSTimeoutError
from nats.js import JetStreamContext
from nats.js.api import ConsumerConfig, AckPolicy, DeliverPolicy

from .logging import get_logger

log = get_logger("nats")

TASKS_STREAM = "ML_TASKS"
RESULTS_STREAM = "ML_RESULTS"
TASKS_SUBJECT = "ml.tasks.>"
DURABLE = "ml-workers"

ACK_WAIT_SECONDS = 10 * 60
MAX_DELIVER = 5


@asynccontextmanager
async def nats_connect(url: str, token: str | None = None) -> AsyncIterator[NATS]:
    nc = await nats.connect(
        url,
        token=token,
        max_reconnect_attempts=-1,
        reconnect_time_wait=2,
        name="pinz-ml-orchestrator",
    )
    try:
        yield nc
    finally:
        await nc.drain()


async def ensure_pull_subscription(
    js: JetStreamContext,
    *,
    subject: str = TASKS_SUBJECT,
    stream: str = TASKS_STREAM,
    durable: str = DURABLE,
):
    """Создаёт/привязывает durable pull-consumer 'ml-workers'.

    Streams и consumer'ы обычно создаются trip-service'ом (см. integration guide §1),
    но мы дополнительно описываем желаемую конфигурацию здесь — на случай локального
    запуска без бэкенда. Если consumer уже существует с другими настройками,
    pull_subscribe просто к нему подключится.
    """
    cfg = ConsumerConfig(
        durable_name=durable,
        ack_policy=AckPolicy.EXPLICIT,
        ack_wait=ACK_WAIT_SECONDS,
        max_deliver=MAX_DELIVER,
        deliver_policy=DeliverPolicy.ALL,
        filter_subject=subject,
    )
    try:
        return await js.pull_subscribe(
            subject=subject,
            durable=durable,
            stream=stream,
            config=cfg,
        )
    except Exception as e:  # noqa: BLE001
        log.warning("pull_subscribe with config failed, fallback to default", error=str(e))
        return await js.pull_subscribe(subject=subject, durable=durable, stream=stream)


HandlerType = Callable[[bytes, Msg], Awaitable[None]]


async def run_pull_loop(
    js: JetStreamContext,
    sub,
    handler: HandlerType,
    *,
    batch: int = 8,
    fetch_timeout: int = 30,
    stop_event: asyncio.Event | None = None,
) -> None:
    """Основной pull-loop"""

    while stop_event is None or not stop_event.is_set():
        try:
            msgs = await sub.fetch(batch=batch, timeout=fetch_timeout)
        except NATSTimeoutError:
            continue
        except asyncio.CancelledError:
            raise
        except Exception as e:  # noqa: BLE001
            log.error("pull fetch failed", error=str(e))
            await asyncio.sleep(2)
            continue

        await asyncio.gather(
            *(_safe_dispatch(handler, m) for m in msgs),
            return_exceptions=False,
        )


async def _safe_dispatch(handler: HandlerType, msg: Msg) -> None:
    try:
        await handler(msg.data, msg)
    except Exception as e:  # noqa: BLE001
        log.error("handler raised, NAK with delay", error=str(e), subject=msg.subject)
        try:
            await msg.nak(delay=10)
        except Exception:  # noqa: BLE001
            pass


async def publish_result(js: JetStreamContext, flow: str, payload: bytes) -> None:
    subject = f"ml.results.{flow}"
    await js.publish(subject, payload, stream=RESULTS_STREAM)


async def ensure_streams_local_dev(js: JetStreamContext) -> None:
    """Создаёт streams ML_TASKS / ML_RESULTS / ML_TASKS_DLQ если их нет.

    Используется только в локальной разработке"""
    from nats.js.api import StreamConfig, RetentionPolicy, StorageType
    from nats.js.errors import NotFoundError

    async def _ensure(name: str, subjects: list[str], *, work_queue: bool = False) -> None:
        try:
            await js.stream_info(name)
        except NotFoundError:
            await js.add_stream(
                StreamConfig(
                    name=name,
                    subjects=subjects,
                    # ML_TASKS — WorkQueue (каждое сообщение обрабатывается одним worker'ом)
                    # ML_RESULTS — Limits (обычный stream: тесты и backend читают независимо)
                    retention=RetentionPolicy.WORK_QUEUE if work_queue else RetentionPolicy.LIMITS,
                    max_age=24 * 60 * 60,
                    storage=StorageType.FILE,
                    num_replicas=1,
                )
            )
            log.info("stream created", name=name, subjects=subjects)

    await _ensure(TASKS_STREAM, ["ml.tasks.>"], work_queue=True)
    await _ensure(RESULTS_STREAM, ["ml.results.>"], work_queue=False)  # NOT WorkQueue
    await _ensure("ML_TASKS_DLQ", ["ml.dlq.>"], work_queue=False)
