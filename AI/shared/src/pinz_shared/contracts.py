"""Контракты NATS-сообщений MLTask / MLResult.

Полностью соответствуют документации Backend:
- docs/ml_integration_guide_python.md (media flows)
- docs/ml_text_moderation_guide_python.md (text_moderation flow)
"""

from __future__ import annotations

from enum import Enum

from pydantic import BaseModel, ConfigDict, Field


class Flow(str, Enum):
    CREATION = "creation"
    ADD_MEDIA = "add_media"
    PIN_UPLOAD_CREATION = "pin_upload.creation"
    PIN_UPLOAD_ADDITION = "pin_upload.addition"
    TEXT_MODERATION = "text_moderation"


class Category(str, Enum):
    SIGHT = "sight"
    NATURE = "nature"
    LEISURE = "leisure"
    HOUSING = "housing"
    FOOD = "food"
    SHOPPING = "shopping"
    TRANSPORT = "transport"
    ENTERTAINMENT = "entertainment"
    EVENT = "event"
    SPORT = "sport"
    WORK = "work"
    CUSTOM = "custom"


# ---------- MLTaskMessage (media flows) ----------


class MediaItem(BaseModel):
    model_config = ConfigDict(extra="ignore")

    media_id: str
    is_new: bool = True
    media_type: str | None = None
    s3_key: str | None = None
    get_url: str
    content_type: str | None = None
    captured_at_unix: int | None = None
    latitude: float | None = None
    longitude: float | None = None


class PinItem(BaseModel):
    model_config = ConfigDict(extra="ignore")

    pin_id: str
    is_new: bool = True
    media: list[MediaItem] = Field(default_factory=list)
    description: str | None = None


class MLTaskMessage(BaseModel):
    model_config = ConfigDict(extra="ignore")

    flow: Flow
    trip_id: str
    session_id: str | None = None
    target_pin_id: str | None = None
    pins: list[PinItem] = Field(default_factory=list)
    expires_at_unix: int | None = None


# ---------- MLTextTaskMessage (text_moderation flow) ----------


class MLTextTaskItem(BaseModel):
    model_config = ConfigDict(extra="ignore")

    item_id: str
    entity_kind: str  # "trip" | "pin"
    entity_id: str
    field: str        # "name" | "description"
    text: str


class MLTextTaskMessage(BaseModel):
    model_config = ConfigDict(extra="ignore")

    flow: Flow = Flow.TEXT_MODERATION
    trip_id: str | None = None
    items: list[MLTextTaskItem]
    expires_at_unix: int | None = None


# ---------- MLResultMessage (unified) ----------


class PinSuggestion(BaseModel):
    model_config = ConfigDict(extra="ignore")

    pin_id: str
    category: Category = Category.CUSTOM
    tags: list[str] = Field(default_factory=list)


class MLTextResult(BaseModel):
    model_config = ConfigDict(extra="ignore")

    item_id: str
    entity_kind: str
    entity_id: str
    field: str
    censored: bool


class MLResultMessage(BaseModel):
    """Единый MLResultMessage. В media-flow заполняется similar/nsfw/pin_suggestions,
    в text_moderation flow — только text_results."""

    model_config = ConfigDict(extra="ignore")

    flow: Flow
    trip_id: str | None = None
    session_id: str | None = None

    similar_groups: list[list[str]] = Field(default_factory=list)
    nsfw_ids: list[str] = Field(default_factory=list)
    pin_suggestions: list[PinSuggestion] = Field(default_factory=list)
    text_results: list[MLTextResult] = Field(default_factory=list)

    def to_json_bytes(self) -> bytes:
        return self.model_dump_json(exclude_none=True).encode()
