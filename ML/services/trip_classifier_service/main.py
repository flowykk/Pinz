from __future__ import annotations

import os
from pathlib import Path
from typing import Optional

import numpy as np
from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title='Trip Classifier Service')


ML_ROOT = Path(__file__).resolve().parents[2]  # .../ML
ARTIFACTS = ML_ROOT / "artifacts"


class TripRequest(BaseModel):
    text: str


class TripResponse(BaseModel):
    trip_type: str
    score: Optional[float] = None
    model: str


def _load_classifier():
    path = ARTIFACTS / "trip_classifier_tfidf_logreg.joblib"
    if not path.exists():
        return None, None
    import joblib

    return joblib.load(path), str(path)


CLF, CLF_PATH = _load_classifier()


def classify_text(text: str) -> TripResponse:
    # 1) Если обученный артефакт есть — используем
    if CLF is not None:
        try:
            pred = CLF.predict([text])[0]
            score = None
            if hasattr(CLF, "predict_proba"):
                proba = CLF.predict_proba([text])[0]
                score = float(np.max(proba))
            return TripResponse(trip_type=str(pred), score=score, model="tfidf_logreg")
        except Exception:
            pass

    # 2) Zero-shot fallback (полезно, когда нет размеченного датасета)
    try:
        from transformers import pipeline

        candidate_labels = ["adventure", "relaxation", "culture", "family", "romantic", "business", "unknown"]
        z = pipeline("zero-shot-classification", model=os.getenv("TRIP_ZS_MODEL", "facebook/bart-large-mnli"))
        out = z(text, candidate_labels=candidate_labels, multi_label=False)
        trip_type = out["labels"][0]
        score = float(out["scores"][0])
        return TripResponse(trip_type=trip_type, score=score, model="zero_shot")
    except Exception:
        # 3) самый простой heuristic
        t = text.lower()
        if any(k in t for k in ["ski", "mountain", "hiking", "climb", "adventure"]):
            return TripResponse(trip_type="adventure", score=1.0, model="heuristic")
        if any(k in t for k in ["beach", "sun", "sea", "relax", "spa"]):
            return TripResponse(trip_type="relaxation", score=1.0, model="heuristic")
        return TripResponse(trip_type="unknown", score=0.0, model="heuristic")


@app.post('/classify')
async def classify(req: TripRequest):
    return classify_text(req.text)

@app.get('/health')
async def health():
    return {'status': 'ok', 'artifact': str(CLF_PATH) if CLF_PATH else None}
