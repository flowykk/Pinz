"""Быстрый smoke-тест: проверяет что все gRPC сервисы отвечают на health-check
и NATS доступен. Запускать после run_local.sh.

    python scripts/smoke_test.py
"""

from __future__ import annotations

import asyncio
import sys
import os

# Добавляем пути
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(ROOT, "proto", "gen"))
sys.path.insert(0, os.path.join(ROOT, "shared", "src"))


async def check_grpc(name: str, addr: str) -> bool:
    import grpc
    from grpc_health.v1 import health_pb2, health_pb2_grpc
    try:
        ch = grpc.aio.insecure_channel(addr)
        stub = health_pb2_grpc.HealthStub(ch)
        resp = await asyncio.wait_for(
            stub.Check(health_pb2.HealthCheckRequest()),
            timeout=5,
        )
        ok = resp.status == health_pb2.HealthCheckResponse.SERVING
        await ch.close()
        return ok
    except Exception as e:
        print(f"  [{name}] FAIL: {e}")
        return False


async def check_nats(url: str) -> bool:
    import nats
    try:
        nc = await asyncio.wait_for(nats.connect(url), timeout=5)
        await nc.drain()
        return True
    except Exception as e:
        print(f"  [NATS] FAIL: {e}")
        return False


async def check_http(name: str, url: str) -> bool:
    import httpx
    try:
        async with httpx.AsyncClient(timeout=5) as client:
            r = await client.get(url)
            return r.status_code < 500
    except Exception as e:
        print(f"  [{name}] FAIL: {e}")
        return False


async def main():
    nats_url = os.environ.get("NATS_URL", "nats://localhost:4222")
    qdrant_url = os.environ.get("QDRANT_URL", "http://localhost:6333")
    meili_url = os.environ.get("MEILI_URL", "http://localhost:7700")

    services = {
        "moderation":  os.environ.get("MODERATION_GRPC", "localhost:50051"),
        "embedding":   os.environ.get("EMBEDDING_GRPC",  "localhost:50052"),
        "classifier":  os.environ.get("CLASSIFIER_GRPC", "localhost:50053"),
        "similarity":  os.environ.get("SIMILARITY_GRPC", "localhost:50054"),
        "search":      os.environ.get("SEARCH_GRPC",     "localhost:50055"),
    }

    results: dict[str, bool] = {}

    print("\n=== Pinz AI Smoke Test ===\n")

    # NATS
    print("Checking NATS...")
    results["NATS"] = await check_nats(nats_url)

    # Qdrant
    print("Checking Qdrant...")
    results["Qdrant"] = await check_http("Qdrant", f"{qdrant_url}/healthz")

    # Meilisearch
    print("Checking Meilisearch...")
    results["Meilisearch"] = await check_http("Meilisearch", f"{meili_url}/health")

    # gRPC services
    print("Checking gRPC services...")
    for name, addr in services.items():
        results[name] = await check_grpc(name, addr)

    print("\n--- Results ---")
    all_ok = True
    for name, ok in results.items():
        status = "✓ OK" if ok else "✗ FAIL"
        print(f"  {status}  {name}")
        if not ok:
            all_ok = False

    print()
    if all_ok:
        print("All systems operational. Ready for integration tests.")
        sys.exit(0)
    else:
        print("Some services are down. Check logs in AI/logs/")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
