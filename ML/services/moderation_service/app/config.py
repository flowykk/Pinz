from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    model_cache_dir: str = "/model_cache"
    text_model_name: str = "cointegrated/rubert-tiny-toxicity"
    text_model_multilingual: str = "unitary/toxic-bert"
    text_toxicity_threshold: float = 0.5
    image_nsfw_threshold: float = 0.6
    device: str = "cpu"

    class Config:
        env_file = ".env"


settings = Settings()
