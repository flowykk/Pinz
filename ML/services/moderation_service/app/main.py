from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.routers.moderation import router as moderation_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    # pre-load models on startup so the first request is not slow
    from app.routers.moderation import get_image_mod, get_text_mod
    get_text_mod()
    get_image_mod()
    yield


app = FastAPI(
    title="Pinz Moderation Service",
    version="0.1.0",
    lifespan=lifespan,
)

app.include_router(moderation_router)


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok"}
