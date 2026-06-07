"""Unit-тесты MLP-головы классификатора"""

import numpy as np
import pytest
import torch

from classifier_service.src.head import PinHead, Classifier, _aggregate


CATEGORIES = ["sight", "nature", "food", "housing", "entertainment", "transport"]
DIM = 768


def test_aggregate_single_emb():
    embs = [np.ones(DIM, dtype=np.float32) * 0.5]
    agg = _aggregate(embs)
    assert agg.shape == (DIM,)
    assert abs(np.linalg.norm(agg) - 1.0) < 1e-5


def test_aggregate_empty():
    agg = _aggregate([])
    assert agg.size == 0


def test_aggregate_multiple_same():
    embs = [np.ones(DIM, dtype=np.float32)] * 4
    agg = _aggregate(embs)
    assert abs(np.linalg.norm(agg) - 1.0) < 1e-5


def test_aggregate_opposite():
    e1 = np.ones(DIM, dtype=np.float32)
    e2 = -np.ones(DIM, dtype=np.float32)
    agg = _aggregate([e1, e2])
    assert agg.shape == (DIM,)


def test_pin_head_output_shape():
    head = PinHead(emb_dim=DIM, n_classes=len(CATEGORIES))
    x = torch.zeros(1, DIM)
    with torch.inference_mode():
        out = head(x)
    assert out.shape == (1, len(CATEGORIES))


def test_pin_head_softmax_sums_to_1():
    head = PinHead(emb_dim=DIM, n_classes=len(CATEGORIES))
    x = torch.randn(4, DIM)
    with torch.inference_mode():
        logits = head(x)
        probs = torch.softmax(logits, dim=-1)
    assert torch.allclose(probs.sum(dim=-1), torch.ones(4), atol=1e-5)


@pytest.fixture
def prototypes_file(tmp_path):
    """Создаём временный файл с фейковыми прототипами категорий."""
    protos = np.random.randn(len(CATEGORIES), DIM).astype(np.float32)
    norms = np.linalg.norm(protos, axis=1, keepdims=True) + 1e-12
    protos /= norms
    path = tmp_path / "protos.npy"
    np.save(path, protos)
    return str(path)


def test_classifier_zeroshot_returns_valid_category(prototypes_file):
    clf = Classifier(
        emb_dim=DIM,
        categories=CATEGORIES,
        head_path="/nonexistent.pt",
        prototypes_path=prototypes_file,
    )
    embs = [np.random.randn(DIM).tolist() for _ in range(3)]
    cat, conf = clf.predict(embs)
    assert cat in CATEGORIES
    assert 0.0 <= conf <= 1.0


def test_classifier_no_weights_returns_custom():
    clf = Classifier(
        emb_dim=DIM,
        categories=CATEGORIES,
        head_path="/nonexistent.pt",
        prototypes_path="/nonexistent.npy",
    )
    cat, conf = clf.predict([np.ones(DIM).tolist()])
    assert cat == "custom"
    assert conf == 0.0


def test_classifier_empty_embeddings_returns_custom(prototypes_file):
    clf = Classifier(
        emb_dim=DIM,
        categories=CATEGORIES,
        head_path="/nonexistent.pt",
        prototypes_path=prototypes_file,
    )
    cat, conf = clf.predict([])
    assert cat == "custom"
    assert conf == 0.0


def test_classifier_with_trained_head(tmp_path):
    """Сохраняем голову с рандомными весами → загружаем → predict работает."""
    head = PinHead(emb_dim=DIM, n_classes=len(CATEGORIES))
    head_path = str(tmp_path / "head.pt")
    torch.save(head.state_dict(), head_path)

    clf = Classifier(
        emb_dim=DIM,
        categories=CATEGORIES,
        head_path=head_path,
        prototypes_path="/nonexistent.npy",
    )
    embs = [np.random.randn(DIM).tolist()]
    cat, conf = clf.predict(embs)
    assert cat in CATEGORIES
    assert 0.0 <= conf <= 1.0
