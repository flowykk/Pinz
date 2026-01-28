from __future__ import annotations

import os
from io import BytesIO
from pathlib import Path
from typing import Optional

import numpy as np
from fastapi import FastAPI, File, UploadFile
from pydantic import BaseModel

app = FastAPI(title='Moderation Service')


ML_ROOT = Path(__file__).resolve().parents[2]  # .../ML
ARTIFACTS = ML_ROOT / "artifacts"


class ModerateRequest(BaseModel):
    text: Optional[str] = None
    image_path: Optional[str] = None


class ModerateResponse(BaseModel):
    text_flag: Optional[bool] = None
    image_flag: Optional[bool] = None
    text_score: Optional[float] = None
    image_score: Optional[float] = None
    text_model: Optional[str] = None
    image_model: Optional[str] = None


def _load_text_baseline():
    """
    Loads sklearn pipeline saved by notebook:
    ML/artifacts/text_baseline_tfidf_logreg.joblib
    """
    path = ARTIFACTS / "text_baseline_tfidf_logreg.joblib"
    if not path.exists():
        return None, None
    import joblib

    model = joblib.load(path)
    return model, str(path)


def _sigmoid(x: float) -> float:
    return float(1.0 / (1.0 + np.exp(-x)))


TEXT_BASELINE, TEXT_BASELINE_PATH = _load_text_baseline()


def moderate_text(text: str) -> tuple[bool, float, str]:
    """
    Returns: (flag, score, model_name)
    score in [0..1] is probability of positive class if available.
    """
    if TEXT_BASELINE is None:
        # fallback heuristic
        t = text.lower()
        bad = any(w in t for w in ["badword", "spam", "scam"])
        return bad, 1.0 if bad else 0.0, "heuristic"

    try:
        proba = None
        if hasattr(TEXT_BASELINE, "predict_proba"):
            proba = TEXT_BASELINE.predict_proba([text])[0]
            score = float(proba[-1])
        else:
            # some estimators only provide decision_function
            score = _sigmoid(float(TEXT_BASELINE.decision_function([text])[0]))
        pred = int(TEXT_BASELINE.predict([text])[0])
        return bool(pred == 1), score, "tfidf_logreg"
    except Exception:
        # last resort fallback
        return False, 0.0, "fallback_error"


def moderate_image_bytes(image_bytes: bytes) -> tuple[bool, float, str]:
    """
    Minimal inference for image model.
    Expects artifact created by notebook:
      ML/artifacts/image_resnet18_best.pt
    """
    ckpt = ARTIFACTS / "image_resnet18_best.pt"
    if not ckpt.exists():
        return False, 0.0, "no_image_model"

    import torch
    from PIL import Image
    from torchvision import models, transforms

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    state = torch.load(ckpt, map_location=device)
    num_labels = int(state.get("num_labels", 2))

    model = models.resnet18(weights=None)
    model.fc = torch.nn.Linear(model.fc.in_features, num_labels)
    model.load_state_dict(state["model_state"])
    model.eval().to(device)

    tf = transforms.Compose(
        [
            transforms.Resize((224, 224)),
            transforms.ToTensor(),
            transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
        ]
    )

    im = Image.open(BytesIO(image_bytes)).convert("RGB")
    x = tf(im).unsqueeze(0).to(device)
    with torch.no_grad():
        logits = model(x)[0].detach().cpu().numpy()
    # score = softmax prob of last class (условно "toxic/flagged")
    exp = np.exp(logits - np.max(logits))
    probs = exp / exp.sum()
    score = float(probs[-1])
    flag = bool(int(np.argmax(probs)) == (num_labels - 1))
    return flag, score, "resnet18"


@app.post('/moderate')
async def moderate(req: ModerateRequest) -> ModerateResponse:
    text_flag = None
    text_score = None
    text_model = None

    image_flag = None
    image_score = None
    image_model = None

    if req.text:
        text_flag, text_score, text_model = moderate_text(req.text)

    if req.image_path:
        p = Path(req.image_path)
        if not p.is_absolute():
            # allow relative paths inside ML/data or current working dir
            p2 = (ML_ROOT / p).resolve()
            if p2.exists():
                p = p2
        if p.exists():
            image_flag, image_score, image_model = moderate_image_bytes(p.read_bytes())
        else:
            image_flag, image_score, image_model = False, 0.0, "image_path_not_found"

    return ModerateResponse(
        text_flag=text_flag,
        image_flag=image_flag,
        text_score=text_score,
        image_score=image_score,
        text_model=text_model,
        image_model=image_model,
    )


@app.post("/moderate/image")
async def moderate_image(file: UploadFile = File(...)) -> ModerateResponse:
    data = await file.read()
    image_flag, image_score, image_model = moderate_image_bytes(data)
    return ModerateResponse(image_flag=image_flag, image_score=image_score, image_model=image_model)


@app.post("/moderate/text")
async def moderate_text_endpoint(req: ModerateRequest) -> ModerateResponse:
    if not req.text:
        return ModerateResponse(text_flag=None, text_score=None, text_model=None)
    text_flag, text_score, text_model = moderate_text(req.text)
    return ModerateResponse(text_flag=text_flag, text_score=text_score, text_model=text_model)

@app.get('/health')
async def health():
    return {'status': 'ok'}
