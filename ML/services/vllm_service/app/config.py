from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # LLaMA Guard 2 8B — open safety model for content moderation via LLM
    model_name: str = "meta-llama/Meta-Llama-Guard-2-8B"
    tensor_parallel_size: int = 1
    max_model_len: int = 4096
    gpu_memory_utilization: float = 0.90
    # generation
    max_new_tokens: int = 128
    temperature: float = 0.0

    class Config:
        env_file = ".env"


settings = Settings()
