"""Интеграционный тест: media-flow creation через реальный NATS.

Публикует задачу в ml.tasks.creation, ждёт ответа из ml.results.creation.
Требует запущенной инфраструктуры (NATS + все микросервисы).

NOTE: NSFW detection и image embeddings требуют Triton Inference Server.
Без Triton эти поля будут пустыми (NSFW = [] и embeddings = []),
но flow должен завершиться без ошибки.
"""

import pytest
import uuid

from tests.helpers.nats_helper import publish_and_wait


VALID_CATEGORIES = {
    "sight", "nature", "leisure", "housing", "food",
    "shopping", "transport", "entertainment", "event",
    "sport", "work", "custom",
}

# Публичная тестовая картинка (пейзаж, не NSFW)
TEST_IMAGE_URL = (
    "https://upload.wikimedia.org/wikipedia/commons/thumb/1/1a/"
    "24701-nature-natural-beauty.jpg/320px-24701-nature-natural-beauty.jpg"
)


def _make_task(*, n_pins: int = 1, n_media_per_pin: int = 1) -> dict:
    pins = []
    for _ in range(n_pins):
        pin_id = str(uuid.uuid4())
        media = [
            {
                "media_id": str(uuid.uuid4()),
                "is_new": True,
                "media_type": "image/jpeg",
                "s3_key": f"trips/test/{uuid.uuid4()}.jpg",
                "get_url": TEST_IMAGE_URL,
                "content_type": "image/jpeg",
                "captured_at_unix": 1700000000,
                "latitude": 48.8566,
                "longitude": 2.3522,
            }
            for _ in range(n_media_per_pin)
        ]
        pins.append({"pin_id": pin_id, "is_new": True, "media": media})

    return {
        "flow": "creation",
        "trip_id": str(uuid.uuid4()),
        "session_id": str(uuid.uuid4()),
        "pins": pins,
        "expires_at_unix": 9999999999,
    }


@pytest.mark.asyncio
@pytest.mark.timeout(90)
async def test_creation_flow_returns_result():
    """Полный цикл: публикация → оркестратор → сервисы → результат."""
    task = _make_task()

    result = await publish_and_wait("creation", task, timeout=80)

    assert result["flow"] == "creation"
    # Обязательные поля всегда присутствуют (даже пустыми)
    assert "similar_groups" in result
    assert "nsfw_ids" in result
    assert "pin_suggestions" in result


@pytest.mark.asyncio
@pytest.mark.timeout(90)
async def test_creation_flow_pin_suggestion_category():
    """Классификатор должен вернуть допустимую категорию."""
    task = _make_task()

    result = await publish_and_wait("creation", task, timeout=80)

    for sug in result.get("pin_suggestions", []):
        assert sug["category"] in VALID_CATEGORIES, (
            f"unexpected category: {sug['category']}"
        )


@pytest.mark.asyncio
@pytest.mark.timeout(30)
async def test_creation_flow_empty_pins():
    """Задача без пинов должна вернуть пустые поля без ошибки."""
    task = {
        "flow": "creation",
        "trip_id": str(uuid.uuid4()),
        "session_id": str(uuid.uuid4()),
        "pins": [],
        "expires_at_unix": 9999999999,
    }

    result = await publish_and_wait("creation", task, timeout=25)

    assert result["flow"] == "creation"
    assert result.get("nsfw_ids", []) == []
    assert result.get("similar_groups", []) == []
    assert result.get("pin_suggestions", []) == []


@pytest.mark.asyncio
@pytest.mark.timeout(90)
async def test_add_media_flow():
    """add_media flow — добавление медиа к существующему пину."""
    task = {
        "flow": "add_media",
        "trip_id": str(uuid.uuid4()),
        "session_id": str(uuid.uuid4()),
        "pins": [
            {
                "pin_id": str(uuid.uuid4()),
                "is_new": False,   # уже существующий пин
                "media": [
                    {
                        "media_id": str(uuid.uuid4()),
                        "is_new": True,  # новое медиа
                        "media_type": "image/jpeg",
                        "s3_key": "trips/test/new.jpg",
                        "get_url": TEST_IMAGE_URL,
                        "content_type": "image/jpeg",
                        "captured_at_unix": 1700000001,
                        "latitude": 48.8566,
                        "longitude": 2.3522,
                    }
                ],
            }
        ],
        "expires_at_unix": 9999999999,
    }

    result = await publish_and_wait("add_media", task, timeout=80)

    assert result["flow"] == "add_media"
    assert "nsfw_ids" in result
    # add_media для существующего пина не должен создавать pin_suggestions
    assert result.get("pin_suggestions", []) == []
