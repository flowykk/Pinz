from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.clients.es_client import ensure_index
from app.clients.qdrant_client import ensure_collection
from app.routers.search import router as search_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    await ensure_index()
    await ensure_collection()
    yield


app = FastAPI(
    title="Pinz Search Service",
    version="0.1.0",
    lifespan=lifespan,
)

app.include_router(search_router)


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok"}
