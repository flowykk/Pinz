from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.routers.embeddings import router as embed_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    from app.routers.embeddings import get_image_enc, get_text_enc
    get_text_enc()
    get_image_enc()
    yield


app = FastAPI(
    title="Pinz Embedding Service",
    version="0.1.0",
    lifespan=lifespan,
)

app.include_router(embed_router)


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok"}
