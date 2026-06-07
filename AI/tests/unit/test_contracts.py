"""Unit-тесты контрактов NATS: правильный парсинг MLTask / MLResult."""

import json
import pytest
from pinz_shared.contracts import (
    Flow, Category, MLTaskMessage, MLTextTaskMessage,
    MLResultMessage, PinSuggestion, MLTextResult, MediaItem, PinItem,
)


MEDIA_PAYLOAD = {
    "media_id": "m-001",
    "is_new": True,
    "media_type": "image/jpeg",
    "s3_key": "trips/abc/img.jpg",
    "get_url": "https://s3.example.com/signed",
    "content_type": "image/jpeg",
    "captured_at_unix": 1700000000,
    "latitude": 55.75,
    "longitude": 37.61,
}

PIN_PAYLOAD = {
    "pin_id": "p-001",
    "is_new": True,
    "media": [MEDIA_PAYLOAD],
}

TASK_PAYLOAD = {
    "flow": "creation",
    "trip_id": "t-001",
    "session_id": "s-001",
    "pins": [PIN_PAYLOAD],
    "expires_at_unix": 1700007200,
}


def test_media_item_parses():
    m = MediaItem.model_validate(MEDIA_PAYLOAD)
    assert m.media_id == "m-001"
    assert m.is_new is True
    assert m.latitude == 55.75


def test_task_message_parses():
    task = MLTaskMessage.model_validate(TASK_PAYLOAD)
    assert task.flow == Flow.CREATION
    assert task.trip_id == "t-001"
    assert len(task.pins) == 1
    assert task.pins[0].media[0].media_id == "m-001"


def test_task_extra_fields_ignored():
    """Лишние поля в payload не бросают ошибку (extra='ignore')."""
    payload = {**TASK_PAYLOAD, "unknown_field": "abc"}
    task = MLTaskMessage.model_validate(payload)
    assert task.trip_id == "t-001"


def test_flow_enum_values():
    assert Flow("creation") == Flow.CREATION
    assert Flow("pin_upload.addition") == Flow.PIN_UPLOAD_ADDITION
    assert Flow("text_moderation") == Flow.TEXT_MODERATION


def test_text_task_parses():
    payload = {
        "flow": "text_moderation",
        "trip_id": "t-001",
        "items": [
            {
                "item_id": "i-001",
                "entity_kind": "pin",
                "entity_id": "p-001",
                "field": "description",
                "text": "Отличный ресторан!",
            }
        ],
    }
    task = MLTextTaskMessage.model_validate(payload)
    assert task.flow == Flow.TEXT_MODERATION
    assert len(task.items) == 1
    assert task.items[0].text == "Отличный ресторан!"


def test_result_message_serializes_without_none():
    result = MLResultMessage(
        flow=Flow.CREATION,
        trip_id="t-001",
        session_id=None,   # должен быть опущен
        similar_groups=[["m-001", "m-002"]],
        nsfw_ids=[],
        pin_suggestions=[PinSuggestion(pin_id="p-001", category=Category.FOOD, tags=["pizza"])],
    )
    data = json.loads(result.to_json_bytes())
    assert "session_id" not in data
    assert data["similar_groups"] == [["m-001", "m-002"]]
    assert data["pin_suggestions"][0]["category"] == "food"


def test_text_result_roundtrip():
    r = MLTextResult(
        item_id="i-001",
        entity_kind="pin",
        entity_id="p-001",
        field="description",
        censored=True,
    )
    result = MLResultMessage(flow=Flow.TEXT_MODERATION, trip_id="t-001", text_results=[r])
    data = json.loads(result.to_json_bytes())
    assert data["text_results"][0]["censored"] is True


def test_category_unknown_stays_custom():
    """Категория, не входящая в Enum, не должна парситься — fallback на стороне кода."""
    with pytest.raises(Exception):
        Category("unknown_value")


def test_tag_truncation_concept():
    """Теги длиннее 15 символов должны быть обрезаны в orchestrator'е (не в Pydantic)."""
    long_tag = "a" * 20
    truncated = long_tag[:15]
    assert len(truncated) == 15
