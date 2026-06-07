"""Интеграционный тест: text_moderation flow."""

import pytest
import uuid

from tests.helpers.nats_helper import publish_and_wait


@pytest.mark.asyncio
@pytest.mark.timeout(60)
async def test_text_moderation_clean_texts():
    """Чистые тексты → censored: false для всех."""
    task = {
        "flow": "text_moderation",
        "trip_id": str(uuid.uuid4()),
        "items": [
            {
                "item_id": str(uuid.uuid4()),
                "entity_kind": "trip",
                "entity_id": str(uuid.uuid4()),
                "field": "name",
                "text": "Amazing trip to the mountains",
            },
            {
                "item_id": str(uuid.uuid4()),
                "entity_kind": "pin",
                "entity_id": str(uuid.uuid4()),
                "field": "description",
                "text": "Beautiful sunset at the beach",
            },
        ],
        "expires_at_unix": 9999999999,
    }

    result = await publish_and_wait("text_moderation", task, timeout=55)

    assert result["flow"] == "text_moderation"
    verdicts = result["text_results"]
    assert len(verdicts) == 2
    for v in verdicts:
        assert v["censored"] is False, (
            f"clean text wrongly censored: {v}"
        )


@pytest.mark.asyncio
@pytest.mark.timeout(60)
async def test_text_moderation_result_has_required_fields():
    """Результат должен содержать все обязательные поля из контракта."""
    item_id = str(uuid.uuid4())
    entity_id = str(uuid.uuid4())
    task = {
        "flow": "text_moderation",
        "trip_id": str(uuid.uuid4()),
        "items": [
            {
                "item_id": item_id,
                "entity_kind": "pin",
                "entity_id": entity_id,
                "field": "description",
                "text": "Beautiful sunset view",
            }
        ],
        "expires_at_unix": 9999999999,
    }

    result = await publish_and_wait("text_moderation", task, timeout=55)

    verdict = result["text_results"][0]
    # Обязательные поля из контракта (ml_text_moderation_guide_python.md §3)
    assert verdict["item_id"] == item_id
    assert verdict["entity_kind"] == "pin"
    assert verdict["entity_id"] == entity_id
    assert verdict["field"] == "description"
    assert "censored" in verdict


@pytest.mark.asyncio
@pytest.mark.timeout(60)
async def test_text_moderation_single_item():
    """Один текст — ровно один вердикт."""
    task = {
        "flow": "text_moderation",
        "trip_id": str(uuid.uuid4()),
        "items": [
            {
                "item_id": str(uuid.uuid4()),
                "entity_kind": "pin",
                "entity_id": str(uuid.uuid4()),
                "field": "name",
                "text": "Колизей на закате",
            }
        ],
        "expires_at_unix": 9999999999,
    }

    result = await publish_and_wait("text_moderation", task, timeout=55)

    assert result["flow"] == "text_moderation"
    assert len(result["text_results"]) == 1


@pytest.mark.asyncio
@pytest.mark.timeout(30)
async def test_text_moderation_empty_items():
    """Пустой список items → пустые text_results без ошибки."""
    task = {
        "flow": "text_moderation",
        "trip_id": str(uuid.uuid4()),
        "items": [],
        "expires_at_unix": 9999999999,
    }

    result = await publish_and_wait("text_moderation", task, timeout=25)

    assert result["flow"] == "text_moderation"
    assert result.get("text_results", []) == []
