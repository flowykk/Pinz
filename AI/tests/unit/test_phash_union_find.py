"""Unit-тесты pHash и Union-Find"""

import io
import pytest
from PIL import Image

from pinz_shared.phash import phash, hamming
from pinz_shared.union_find import UnionFind


# ---------- pHash -------------------------------------------------------

import numpy as np


def _make_image(color: tuple[int, int, int], size: int = 224) -> Image.Image:
    return Image.new("RGB", (size, size), color=color)


def _make_gradient(size: int = 224, start: int = 0, end: int = 255) -> Image.Image:
    """Картинка с горизонтальным градиентом — имеет богатый DCT-спектр."""
    arr = np.zeros((size, size, 3), dtype=np.uint8)
    arr[:, :, 0] = np.linspace(start, end, size, dtype=np.uint8)
    arr[:, :, 1] = np.linspace(end, start, size, dtype=np.uint8)
    arr[:, :, 2] = 128
    return Image.fromarray(arr)


def _make_noise(seed: int = 0, size: int = 224) -> Image.Image:
    """Случайный шум — непохожий на другой шум с другим seed'ом."""
    rng = np.random.default_rng(seed)
    arr = rng.integers(0, 256, (size, size, 3), dtype=np.uint8)
    return Image.fromarray(arr)


def test_phash_same_image():
    img = _make_gradient()
    assert phash(img) == phash(img)


def test_phash_identical_images_zero_hamming():
    img = _make_gradient()
    # Одинаковые изображения должны давать нулевое расстояние
    assert hamming(phash(img), phash(img)) == 0


def test_phash_very_different_images_large_hamming():
    """Два разных шумовых изображения должны давать большое расстояние."""
    img1 = _make_noise(seed=1)
    img2 = _make_noise(seed=42)
    # Разные текстуры → разные pHash
    assert hamming(phash(img1), phash(img2)) > 10


def test_phash_similar_images_small_hamming():
    """Почти одинаковые изображения (слегка другой градиент) — малое расстояние."""
    img1 = _make_gradient(start=0, end=200)
    img2 = _make_gradient(start=0, end=205)   # очень похожий градиент
    assert hamming(phash(img1), phash(img2)) <= 10


def test_hamming_is_symmetric():
    a, b = 0b1010101, 0b1100110
    assert hamming(a, b) == hamming(b, a)


def test_hamming_with_zero():
    x = 0b11111111
    assert hamming(x, 0) == bin(x).count("1")


# ---------- Union-Find --------------------------------------------------

def test_uf_no_unions():
    uf = UnionFind(["a", "b", "c"])
    groups = uf.groups(min_size=2)
    assert groups == []


def test_uf_single_pair():
    uf = UnionFind(["a", "b", "c"])
    uf.union("a", "b")
    groups = uf.groups(min_size=2)
    assert len(groups) == 1
    assert sorted(groups[0]) == ["a", "b"]


def test_uf_chain():
    """a–b–c → один кластер."""
    uf = UnionFind(["a", "b", "c", "d"])
    uf.union("a", "b")
    uf.union("b", "c")
    groups = uf.groups(min_size=2)
    assert len(groups) == 1
    assert sorted(groups[0]) == ["a", "b", "c"]


def test_uf_two_separate_groups():
    uf = UnionFind(["a", "b", "c", "d"])
    uf.union("a", "b")
    uf.union("c", "d")
    groups = uf.groups(min_size=2)
    assert len(groups) == 2
    pair1 = sorted(groups[0])
    pair2 = sorted(groups[1])
    assert sorted([pair1, pair2]) == [["a", "b"], ["c", "d"]]


def test_uf_self_union():
    uf = UnionFind(["a", "b"])
    uf.union("a", "a")  # не должно кидать и не должно создавать пары
    assert uf.groups(min_size=2) == []


def test_uf_redundant_union():
    uf = UnionFind(["a", "b", "c"])
    uf.union("a", "b")
    uf.union("a", "b")  # повторно
    groups = uf.groups(min_size=2)
    assert len(groups) == 1
    assert sorted(groups[0]) == ["a", "b"]


def test_uf_min_size_3():
    uf = UnionFind(["a", "b", "c"])
    uf.union("a", "b")
    uf.union("a", "c")
    groups = uf.groups(min_size=3)
    assert len(groups) == 1
    assert sorted(groups[0]) == ["a", "b", "c"]
