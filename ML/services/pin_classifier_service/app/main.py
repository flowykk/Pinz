from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.routers.classify import router as classify_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    from app.routers.classify import get_classifier
    get_classifier()
    yield


app = FastAPI(
    title="Pinz Pin Classifier Service",
    version="0.1.0",
    description="EfficientNet-B4 fine-tuned for pin category classification.",
    lifespan=lifespan,
)

app.include_router(classify_router)


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok"}
