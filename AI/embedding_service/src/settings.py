from pinz_shared import BaseServiceSettings


class Settings(BaseServiceSettings):
    service_name: str = "embedding-service"
    grpc_port: int = 50052

    siglip_model_id: str = "google/siglip-base-patch16-224"
    siglip_image_model: str = "siglip_vision"
    siglip_text_model: str = "siglip_text"
    embedding_dim: int = 768

    qdrant_collection_images: str = "pin_images"
    qdrant_collection_text: str = "pin_texts"
