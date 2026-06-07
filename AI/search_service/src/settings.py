from pinz_shared import BaseServiceSettings


class Settings(BaseServiceSettings):
    service_name: str = "search-service"
    grpc_port: int = 50055

    meili_index: str = "pins"
    qdrant_collection_images: str = "pin_images"
    qdrant_collection_text: str = "pin_texts"
    embedding_dim: int = 768

    rrf_k: int = 60
    default_top_k: int = 10
