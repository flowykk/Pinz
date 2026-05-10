from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    elasticsearch_url: str = "http://localhost:9200"
    elasticsearch_index_pins: str = "pins"
    qdrant_url: str = "http://localhost:6333"
    qdrant_collection_pins: str = "pins"
    embedding_service_url: str = "http://localhost:8002"
    # RRF window size and rank constant
    rrf_k: int = 60
    rrf_window: int = 100
    # embedding dimensions must match embedding_service config
    vector_dim_text: int = 1024
    vector_dim_image: int = 768

    class Config:
        env_file = ".env"


settings = Settings()
