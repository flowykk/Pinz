from fastapi import APIRouter, File, Form, HTTPException, UploadFile
from pydantic import BaseModel

from app.models.image_moderator import ImageModerator
from app.models.text_moderator import TextModerator

router = APIRouter(prefix="/moderate", tags=["moderation"])

_text_mod: TextModerator | None = None
_image_mod: ImageModerator | None = None


def get_text_mod() -> TextModerator:
    global _text_mod
    if _text_mod is None:
        _text_mod = TextModerator()
    return _text_mod


def get_image_mod() -> ImageModerator:
    global _image_mod
    if _image_mod is None:
        _image_mod = ImageModerator()
    return _image_mod


class TextRequest(BaseModel):
    text: str


class ModerationResult(BaseModel):
    flagged: bool
    score: float
    label: str


@router.post("/text", response_model=ModerationResult)
def moderate_text(body: TextRequest) -> ModerationResult:
    if not body.text.strip():
        raise HTTPException(status_code=422, detail="text must not be empty")
    result = get_text_mod().predict(body.text)
    return ModerationResult(
        flagged=result["toxic"],
        score=result["score"],
        label="toxic" if result["toxic"] else "safe",
    )


@router.post("/image", response_model=ModerationResult)
async def moderate_image(file: UploadFile = File(...)) -> ModerationResult:
    if not file.content_type or not file.content_type.startswith("image/"):
        raise HTTPException(status_code=422, detail="uploaded file must be an image")
    data = await file.read()
    result = get_image_mod().predict(data)
    return ModerationResult(
        flagged=result["nsfw"],
        score=result["score"],
        label="nsfw" if result["nsfw"] else "safe",
    )
