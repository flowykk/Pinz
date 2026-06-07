from pinz_shared import BaseServiceSettings


class Settings(BaseServiceSettings):
    service_name: str = "moderation-service"
    grpc_port: int = 50051

    # Triton model name for NSFW classifier
    nsfw_model_name: str = "nsfw_detector"

    toxic_model_id: str = "textdetox/xlmr-large-toxicity-classifier"
    toxic_device: str = "cpu" 
    toxic_max_len: int = 256
