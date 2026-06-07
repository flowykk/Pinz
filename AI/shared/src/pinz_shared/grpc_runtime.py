"""Утилиты для подъёма gRPC-сервера в каждом микросервисе."""

from __future__ import annotations

import asyncio
import signal
from collections.abc import Awaitable, Callable

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

from .logging import get_logger

log = get_logger("grpc")


async def serve_grpc(
    port: int,
    register: Callable[[grpc.aio.Server], Awaitable[None] | None],
    *,
    max_workers: int = 32,
    service_name: str = "pinz-ai",
) -> None:
    server = grpc.aio.server(
        options=[
            ("grpc.max_send_message_length", 64 * 1024 * 1024),
            ("grpc.max_receive_message_length", 64 * 1024 * 1024),
        ]
    )

    res = register(server)
    if asyncio.iscoroutine(res):
        await res

    health_servicer = health.HealthServicer()
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set(service_name, health_pb2.HealthCheckResponse.SERVING)
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)

    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    log.info("grpc server started", port=port, service=service_name)

    stop = asyncio.Event()

    def _on_signal() -> None:
        log.info("shutdown signal received")
        stop.set()

    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, _on_signal)
        except NotImplementedError:
            pass

    await stop.wait()
    await server.stop(grace=10)
