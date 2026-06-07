"""Unit-тесты ToxicClassifier.
"""

import pytest

pytestmark = pytest.mark.slow


@pytest.fixture(scope="module")
def toxic_clf():
    """Загружаем модель один раз на весь модуль."""
    from moderation_service.src.toxic import ToxicClassifier
    return ToxicClassifier(
        model_id="textdetox/xlmr-large-toxicity-classifier",
        device="cpu",
        max_len=128,
        threshold=0.7,
    )


@pytest.mark.asyncio
async def test_clean_text_en(toxic_clf):
    texts = ["I love this beautiful city!"]
    probs = await toxic_clf.predict(texts)
    assert len(probs) == 1
    assert probs[0] < 0.5  # чистый текст → low prob


@pytest.mark.asyncio
async def test_clean_text_ru(toxic_clf):
    texts = ["Красивый закат над горами"]
    probs = await toxic_clf.predict(texts)
    assert probs[0] < 0.5


@pytest.mark.asyncio
async def test_toxic_text_en(toxic_clf):
    texts = ["You are an absolute idiot and I hate you!"]
    probs = await toxic_clf.predict(texts)
    assert probs[0] > 0.5  # токсичный текст


@pytest.mark.asyncio
async def test_batch_mixed(toxic_clf):
    texts = [
        "Amazing restaurant with great food",
        "I will destroy you and your family",
        "Lovely walk in the park",
    ]
    probs = await toxic_clf.predict(texts)
    assert len(probs) == 3
    assert probs[0] < 0.5   # clean
    assert probs[1] > 0.5   # toxic
    assert probs[2] < 0.5   # clean


@pytest.mark.asyncio
async def test_is_toxic_flag(toxic_clf):
    assert not toxic_clf.is_toxic(0.3)
    assert not toxic_clf.is_toxic(0.69)
    assert toxic_clf.is_toxic(0.7)
    assert toxic_clf.is_toxic(0.99)


@pytest.mark.asyncio
async def test_empty_batch(toxic_clf):
    probs = await toxic_clf.predict([])
    assert probs == []
