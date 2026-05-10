from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.routers.inference import router as inference_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    # warm up engine on startup
    from app.engine import get_engine
    get_engine()
    yield


app = FastAPI(
    title="Pinz vLLM Inference Service",
    version="0.1.0",
    description="LLaMA Guard 2 8B via vLLM for LLM-based content safety.",
    lifespan=lifespan,
)

app.include_router(inference_router)


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok"}
