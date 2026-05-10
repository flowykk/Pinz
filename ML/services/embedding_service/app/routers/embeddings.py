from fastapi import APIRouter, File, HTTPException, UploadFile
from pydantic import BaseModel

from app.models.image_encoder import ImageEncoder
from app.models.text_encoder import TextEncoder

router = APIRouter(prefix="/embed", tags=["embeddings"])

_text_enc: TextEncoder | None = None
_image_enc: ImageEncoder | None = None


def get_text_enc() -> TextEncoder:
    global _text_enc
    if _text_enc is None:
        _text_enc = TextEncoder()
    return _text_enc


def get_image_enc() -> ImageEncoder:
    global _image_enc
    if _image_enc is None:
        _image_enc = ImageEncoder()
    return _image_enc


class TextEmbedRequest(BaseModel):
    texts: list[str]
    is_query: bool = False


class EmbedResponse(BaseModel):
    embeddings: list[list[float]]
    dim: int


@router.post("/text", response_model=EmbedResponse)
def embed_text(body: TextEmbedRequest) -> EmbedResponse:
    if not body.texts:
        raise HTTPException(status_code=422, detail="texts must not be empty")
    vecs = get_text_enc().encode(body.texts, is_query=body.is_query)
    return EmbedResponse(embeddings=vecs, dim=len(vecs[0]))


@router.post("/image", response_model=EmbedResponse)
async def embed_image(files: list[UploadFile] = File(...)) -> EmbedResponse:
    raw_images = []
    for f in files:
        if not f.content_type or not f.content_type.startswith("image/"):
            raise HTTPException(status_code=422, detail=f"{f.filename} is not an image")
        raw_images.append(await f.read())
    vecs = get_image_enc().encode(raw_images)
    return EmbedResponse(embeddings=vecs, dim=len(vecs[0]))
