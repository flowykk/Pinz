from pydantic_settings import BaseSettings, SettingsConfigDict


class BaseServiceSettings(BaseSettings):
    """Базовые настройки, наследуемые всеми микросервисами."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    log_level: str = "INFO"
    service_name: str = "pinz-ai"

    nats_url: str = "nats://nats:4222"
    nats_token: str | None = None

    triton_grpc_url: str = "triton:8001"

    moderation_grpc: str = "moderation_service:50051"
    embedding_grpc: str = "embedding_service:50052"
    classifier_grpc: str = "classifier_service:50053"
    similarity_grpc: str = "similarity_service:50054"
    search_grpc: str = "search_service:50055"

    qdrant_url: str = "http://qdrant:6333"
    meili_url: str = "http://meilisearch:7700"
    meili_master_key: str = "pinz-dev-master-key"

    http_fetch_timeout: int = 30
    http_fetch_concurrency: int = 16

    nsfw_threshold: float = 0.7
    toxic_threshold: float = 0.7
    similarity_phash_hamming_max: int = 8
    similarity_cosine_min: float = 0.92
