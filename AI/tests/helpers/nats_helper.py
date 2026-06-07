"""Утилиты для интеграционных тестов: публикация задач и чтение результатов через NATS.

Дизайн:
  - publish_and_wait():
      1. Запоминаем последний seq в ML_RESULTS (ПЕРЕД публикацией).
      2. Публикуем задачу в ML_TASKS.
      3. Pull-consumer с DeliverPolicy.BY_START_SEQUENCE от (last_seq+1)
         гарантирует: не пропустим результат даже если оркестратор обработал
         задачу в том же batch'е, что и предыдущую.
      4. Читаем сообщения и фильтруем по trip_id — skip чужих результатов.

  - ML_RESULTS — Limits stream (не WorkQueue): несколько consumer'ов параллельно.
  - ML_TASKS   — WorkQueue stream: каждая задача потребляется одним воркером.
"""

from __future__ import annotations

import asyncio
import json
import os
import uuid

import nats
from nats.js.api import ConsumerConfig, DeliverPolicy, AckPolicy
from nats.errors import TimeoutError as NATSTimeout


_TASKS_STREAM = "ML_TASKS"
_RESULTS_STREAM = "ML_RESULTS"


def _nats_url() -> str:
    return os.environ.get("NATS_URL", "nats://localhost:4222")


async def publish_and_wait(
    flow: str,
    payload: dict,
    *,
    timeout: float = 60.0,
    nats_url: str | None = None,
) -> dict:
    """Публикует задачу и ждёт соответствующего результата (по trip_id).

    1. Снимаем last_seq стрима ML_RESULTS — чтобы не пропустить результат
       если оркестратор очень быстро обработал задачу.
    2. Публикуем задачу.
    3. Pull-consumer с BY_START_SEQUENCE (last_seq+1) читает все новые
       результаты и ищет совпадение по trip_id.
    """
    url = nats_url or _nats_url()
    trip_id: str = payload.get("trip_id", "")

    nc = await nats.connect(url, max_reconnect_attempts=3)
    js = nc.jetstream()

    try:
        # --- 1. Запоминаем позицию стрима ДО публикации ---
        try:
            info = await js.stream_info(_RESULTS_STREAM)
            start_seq = info.state.last_seq + 1
        except Exception:  # stream пустой или не существует
            start_seq = 1

        # --- 2. Публикуем задачу ---
        await js.publish(
            f"ml.tasks.{flow}",
            json.dumps(payload).encode(),
            stream=_TASKS_STREAM,
        )

        # --- 3. Pull-consumer от start_seq ---
        durable = f"test-{flow}-{uuid.uuid4().hex[:8]}"
        result_subject = f"ml.results.{flow}"

        cfg = ConsumerConfig(
            durable_name=durable,
            filter_subject=result_subject,
            deliver_policy=DeliverPolicy.BY_START_SEQUENCE,
            opt_start_seq=start_seq,
            ack_policy=AckPolicy.EXPLICIT,
            ack_wait=120,
        )
        sub = await js.pull_subscribe(
            subject=result_subject,
            durable=durable,
            stream=_RESULTS_STREAM,
            config=cfg,
        )

        deadline = asyncio.get_event_loop().time() + timeout
        try:
            while asyncio.get_event_loop().time() < deadline:
                remaining = deadline - asyncio.get_event_loop().time()
                if remaining <= 0:
                    break

                try:
                    msgs = await sub.fetch(1, timeout=min(remaining, 5.0))
                except NATSTimeout:
                    continue
                except Exception as e:
                    raise RuntimeError(f"fetch error: {e}") from e

                for msg in msgs:
                    data = json.loads(msg.data)
                    await msg.ack()
                    # Фильтруем: берём только результат нашей задачи
                    if not trip_id or data.get("trip_id") == trip_id:
                        return data
                    # else: чужой результат — пропускаем, ждём дальше

            raise TimeoutError(
                f"No result for flow '{flow}' (trip_id={trip_id}) within {timeout}s. "
                "Check ml_orchestrator logs: tail -f logs/ml_orchestrator.log"
            )
        finally:
            try:
                await sub.unsubscribe()
            except Exception:  # noqa: BLE001
                pass

    finally:
        await nc.drain()


# ---------------------------------------------------------------------------
# Публикация без ожидания результата (для тестов, которым он не нужен)
# ---------------------------------------------------------------------------

async def publish_task(
    flow: str,
    payload: dict,
    *,
    nats_url: str | None = None,
) -> None:
    """Публикует задачу без ожидания результата."""
    url = nats_url or _nats_url()
    nc = await nats.connect(url, max_reconnect_attempts=3)
    js = nc.jetstream()
    try:
        await js.publish(
            f"ml.tasks.{flow}",
            json.dumps(payload).encode(),
            stream=_TASKS_STREAM,
        )
    finally:
        await nc.drain()
