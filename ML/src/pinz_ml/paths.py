from __future__ import annotations

from pathlib import Path


def ml_root() -> Path:
    """
    Returns ML folder path.
    Works whether you run from repo root or ML/ itself.
    """
    here = Path(__file__).resolve()
    # .../ML/src/pinz_ml/paths.py -> .../ML
    ml = here.parents[2]
    return ml


def data_dir() -> Path:
    return ml_root() / "data"


def artifacts_dir() -> Path:
    p = ml_root() / "artifacts"
    p.mkdir(parents=True, exist_ok=True)
    return p

