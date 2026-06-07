from pinz_shared import BaseServiceSettings


class Settings(BaseServiceSettings):
    service_name: str = "classifier-service"
    grpc_port: int = 50053

    embedding_dim: int = 768
    categories: tuple[str, ...] = (
        "sight", "nature", "food", "housing", "entertainment", "transport",
    )
    head_weights: str = "/models/classifier_head.pt"
    zeroshot_prototypes: str = "/models/category_prototypes.npy"

    min_confidence: float = 0.35
