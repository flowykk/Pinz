"""Простой 64-битный perceptual hash на DCT (без внешних зависимостей кроме PIL/numpy)."""

from __future__ import annotations

import numpy as np
from PIL import Image
from scipy.fftpack import dct


def phash(img: Image.Image, hash_size: int = 8, highfreq_factor: int = 4) -> int:
    img_size = hash_size * highfreq_factor
    g = img.convert("L").resize((img_size, img_size), Image.Resampling.LANCZOS)
    a = np.asarray(g, dtype=np.float32)
    d = dct(dct(a, axis=0, norm="ortho"), axis=1, norm="ortho")
    block = d[:hash_size, :hash_size]
    med = np.median(block[1:].flatten() if block.size > 1 else block)
    bits = block > med
    out = 0
    for b in bits.flatten():
        out = (out << 1) | int(b)
    return out


def hamming(a: int, b: int) -> int:
    return (a ^ b).bit_count()
