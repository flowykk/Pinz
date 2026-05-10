from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    model_cache_dir: str = "/model_cache"
    # intfloat/multilingual-e5-large — 560M params, 1024-dim, supports RU/EN
    text_embed_model: str = "intfloat/multilingual-e5-large"
    # OpenAI CLIP ViT-L/14 — 768-dim image embeddings
    image_embed_model: str = "ViT-L-14"
    image_embed_pretrained: str = "openai"
    embedding_dim_text: int = 1024
    embedding_dim_image: int = 768
    device: str = "cpu"

    class Config:
        env_file = ".env"


settings = Settings()
