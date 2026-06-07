"""Класс Union-Find для группировки похожих медиа."""

from __future__ import annotations


class UnionFind:
    def __init__(self, items: list[str]) -> None:
        self._parent: dict[str, str] = {x: x for x in items}
        self._rank: dict[str, int] = {x: 0 for x in items}

    def find(self, x: str) -> str:
        while self._parent[x] != x:
            self._parent[x] = self._parent[self._parent[x]]
            x = self._parent[x]
        return x

    def union(self, a: str, b: str) -> None:
        ra, rb = self.find(a), self.find(b)
        if ra == rb:
            return
        if self._rank[ra] < self._rank[rb]:
            ra, rb = rb, ra
        self._parent[rb] = ra
        if self._rank[ra] == self._rank[rb]:
            self._rank[ra] += 1

    def groups(self, *, min_size: int = 2) -> list[list[str]]:
        buckets: dict[str, list[str]] = {}
        for x in self._parent:
            buckets.setdefault(self.find(x), []).append(x)
        return [sorted(g) for g in buckets.values() if len(g) >= min_size]
