from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # path to fine-tuned EfficientNet-B4 checkpoint
    model_weights_path: str = "/weights/pin_efficientnet_b4.pt"
    num_classes: int = 50
    top_k: int = 5
    device: str = "cpu"

    class Config:
        env_file = ".env"


settings = Settings()
