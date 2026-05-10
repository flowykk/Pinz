from fastapi import APIRouter, File, HTTPException, UploadFile
from pydantic import BaseModel

from app.models.classifier import PinClassifier

router = APIRouter(prefix="/classify", tags=["classification"])

_classifier: PinClassifier | None = None


def get_classifier() -> PinClassifier:
    global _classifier
    if _classifier is None:
        _classifier = PinClassifier()
    return _classifier


class ClassLabel(BaseModel):
    class_id: int
    score: float


class ClassifyResponse(BaseModel):
    top_k: list[ClassLabel]


@router.post("/pin", response_model=ClassifyResponse)
async def classify_pin(file: UploadFile = File(...)) -> ClassifyResponse:
    if not file.content_type or not file.content_type.startswith("image/"):
        raise HTTPException(status_code=422, detail="uploaded file must be an image")
    data = await file.read()
    predictions = get_classifier().predict(data)
    return ClassifyResponse(top_k=[ClassLabel(**p) for p in predictions])
