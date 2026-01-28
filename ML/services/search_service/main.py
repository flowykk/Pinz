from __future__ import annotations

import os
from pathlib import Path
from typing import Any, Optional

import pandas as pd
from fastapi import FastAPI
from pydantic import BaseModel
from sklearn.feature_extraction.text import TfidfVectorizer
from sklearn.metrics.pairwise import cosine_similarity

app = FastAPI(title='Search Service')


class SearchRequest(BaseModel):
    query: str
    k: int = 10

ML_ROOT = Path(__file__).resolve().parents[2]  # .../ML
DATASET_PATH = Path(os.getenv("TRIPS_PARQUET", str(ML_ROOT / "data" / "trips.parquet")))


def _normalize_trip_row(row: dict) -> dict:
    return {
        "id": row.get("id"),
        "title": str(row.get("title", "")),
        "description": str(row.get("description", "")),
        "tags": row.get("tags", []),
    }


class TripIndex:
    def __init__(self):
        self.df: Optional[pd.DataFrame] = None
        self.vectorizer: Optional[TfidfVectorizer] = None
        self.matrix = None

    def is_ready(self) -> bool:
        return self.df is not None and self.vectorizer is not None and self.matrix is not None

    def build_from_parquet(self, path: Path):
        df = pd.read_parquet(path)
        # expected columns: id, title, description, tags (tags may be list or string)
        if "title" not in df.columns:
            raise ValueError(f"trips.parquet must contain 'title' column. got: {list(df.columns)}")
        if "description" not in df.columns:
            df["description"] = ""
        if "tags" not in df.columns:
            df["tags"] = ""

        def to_text(r) -> str:
            tags = r["tags"]
            if isinstance(tags, (list, tuple)):
                tags_s = " ".join(map(str, tags))
            else:
                tags_s = str(tags)
            return f"{r['title']} {r['description']} {tags_s}".strip()

        df = df.copy()
        df["_doc"] = df.apply(to_text, axis=1)
        vec = TfidfVectorizer(ngram_range=(1, 2), max_features=200_000, min_df=1)
        mat = vec.fit_transform(df["_doc"])

        self.df = df
        self.vectorizer = vec
        self.matrix = mat

    def search(self, query: str, k: int) -> list[dict[str, Any]]:
        if not self.is_ready():
            return []
        qv = self.vectorizer.transform([query])
        sims = cosine_similarity(qv, self.matrix)[0]
        k = max(1, min(int(k), 50))
        top_idx = sims.argsort()[::-1][:k]
        out = []
        for i in top_idx:
            r = self.df.iloc[int(i)]
            out.append(
                {
                    "id": r.get("id", int(i)),
                    "title": r.get("title", ""),
                    "description": r.get("description", ""),
                    "tags": r.get("tags", []),
                    "score": float(sims[int(i)]),
                }
            )
        return out


INDEX = TripIndex()


@app.post('/search')
async def search(req: SearchRequest):
    if not INDEX.is_ready():
        # lazy-load
        if DATASET_PATH.exists():
            INDEX.build_from_parquet(DATASET_PATH)
        else:
            # fallback small index
            INDEX.df = pd.DataFrame(
                [
                    {"id": 1, "title": "Sunny beach escape", "description": "", "tags": ["beach", "relaxation"]},
                    {"id": 2, "title": "Mountain skiing trip", "description": "", "tags": ["ski", "adventure"]},
                ]
            )
            INDEX.build_from_parquet = lambda _: None  # type: ignore
            INDEX.vectorizer = TfidfVectorizer(ngram_range=(1, 2), max_features=50_000, min_df=1)
            INDEX.matrix = INDEX.vectorizer.fit_transform(
                (INDEX.df["title"].astype(str) + " " + INDEX.df["tags"].astype(str)).tolist()
            )

    results = INDEX.search(req.query, req.k)
    return {'results': results, "source": str(DATASET_PATH) if DATASET_PATH.exists() else "fallback"}

@app.get('/health')
async def health():
    return {'status': 'ok', 'index_ready': INDEX.is_ready(), 'dataset': str(DATASET_PATH)}
