"""Unit-тесты функции RRF (Reciprocal Rank Fusion) в Search Service."""

import pytest
from search_service.src.server import rrf


def test_rrf_single_list():
    """Один список — просто ранжирование по позиции."""
    rankings = [[("pin-1", "food"), ("pin-2", "nature"), ("pin-3", "sight")]]
    result = rrf(rankings, k=60)
    ids = [r[0] for r in result]
    # Первый должен иметь наибольший score
    assert ids[0] == "pin-1"
    assert result[0][1] > result[1][1] > result[2][1]


def test_rrf_two_lists_consensus():
    """Пин, стоящий первым в обоих списках, должен получить наивысший fused score."""
    r1 = [("pin-A", "food"), ("pin-B", "sight"), ("pin-C", "nature")]
    r2 = [("pin-A", "food"), ("pin-C", "nature"), ("pin-B", "sight")]
    result = rrf([r1, r2], k=60)
    assert result[0][0] == "pin-A"


def test_rrf_three_lists():
    r1 = [("a", "food"), ("b", "sight"), ("c", "nature")]
    r2 = [("c", "nature"), ("a", "food"), ("b", "sight")]
    r3 = [("b", "sight"), ("c", "nature"), ("a", "food")]
    result = rrf([r1, r2, r3], k=60)
    ids = {r[0] for r in result}
    assert ids == {"a", "b", "c"}


def test_rrf_empty_lists():
    result = rrf([[], [], []], k=60)
    assert result == []


def test_rrf_partial_overlap():
    """Пин только в одном списке всё равно попадает в результат."""
    r1 = [("pin-X", "sight")]
    r2 = [("pin-Y", "food")]
    result = rrf([r1, r2], k=60)
    ids = {r[0] for r in result}
    assert "pin-X" in ids
    assert "pin-Y" in ids


def test_rrf_category_non_custom_wins():
    """Если у пина в одном списке 'custom', а в другом 'food' — выбираем 'food'."""
    r1 = [("pin-1", "custom")]
    r2 = [("pin-1", "food")]
    result = rrf([r1, r2], k=60)
    assert result[0][2] == "food"


def test_rrf_scores_are_positive():
    r = [[("a", "sight"), ("b", "food")]]
    result = rrf(r, k=60)
    for _, score, _ in result:
        assert score > 0


def test_rrf_k_affects_scores():
    """k=1 должен давать более высокие score, чем k=100."""
    r = [[("a", "sight"), ("b", "food")]]
    res_low_k = rrf(r, k=1)
    res_high_k = rrf(r, k=100)
    assert res_low_k[0][1] > res_high_k[0][1]
