from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Literal, Optional

import pandas as pd


@dataclass(frozen=True)
class TextDatasetSpec:
    path: Path
    text_col: str = "text"
    label_col: str = "label"
    id_col: Optional[str] = None
    split_col: Optional[str] = None  # e.g. "split" with values train/val/test


@dataclass(frozen=True)
class ImageDatasetSpec:
    path: Path
    label_col: str = "label"
    image_path_col: str = "image_path"
    image_bytes_col: str = "image_bytes"
    id_col: Optional[str] = None
    split_col: Optional[str] = None


def _require_cols(df: pd.DataFrame, cols: Iterable[str], *, name: str) -> None:
    missing = [c for c in cols if c not in df.columns]
    if missing:
        raise ValueError(f"{name}: missing columns: {missing}. Available: {list(df.columns)}")


def load_text_dataset(spec: TextDatasetSpec) -> pd.DataFrame:
    df = pd.read_parquet(spec.path)
    _require_cols(df, [spec.text_col, spec.label_col], name=f"text_dataset({spec.path.name})")
    df = df.copy()
    df[spec.text_col] = df[spec.text_col].astype(str)
    df[spec.label_col] = df[spec.label_col].astype(int)
    return df


def load_image_dataset(spec: ImageDatasetSpec) -> pd.DataFrame:
    df = pd.read_parquet(spec.path)
    if spec.image_path_col not in df.columns and spec.image_bytes_col not in df.columns:
        raise ValueError(
            f"images_dataset({spec.path.name}): expected '{spec.image_path_col}' or '{spec.image_bytes_col}' column"
        )
    if spec.label_col in df.columns:
        df = df.copy()
        df[spec.label_col] = df[spec.label_col].astype(int)
    return df


def split_by_column(
    df: pd.DataFrame,
    *,
    split_col: str,
    train_value: str = "train",
    val_value: str = "val",
    test_value: str = "test",
) -> tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame]:
    _require_cols(df, [split_col], name="split_by_column")
    train = df[df[split_col] == train_value].copy()
    val = df[df[split_col] == val_value].copy()
    test = df[df[split_col] == test_value].copy()
    return train, val, test


def random_split(
    df: pd.DataFrame,
    *,
    train_frac: float = 0.8,
    val_frac: float = 0.1,
    seed: int = 42,
) -> tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame]:
    if not (0 < train_frac < 1) or not (0 < val_frac < 1) or (train_frac + val_frac) >= 1:
        raise ValueError("random_split: require 0<train_frac<1, 0<val_frac<1, train+val < 1")
    df = df.sample(frac=1.0, random_state=seed).reset_index(drop=True)
    n = len(df)
    n_train = int(n * train_frac)
    n_val = int(n * val_frac)
    train = df.iloc[:n_train].copy()
    val = df.iloc[n_train : n_train + n_val].copy()
    test = df.iloc[n_train + n_val :].copy()
    return train, val, test

