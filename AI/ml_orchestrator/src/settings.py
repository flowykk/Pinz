from pinz_shared import BaseServiceSettings


class Settings(BaseServiceSettings):
    service_name: str = "ml-orchestrator"
    # Если true — при старте создадим streams ML_TASKS/ML_RESULTS/ML_TASKS_DLQ
    bootstrap_streams_local: bool = True
    # batch size для fetch из NATS
    pull_batch: int = 8
    pull_timeout: int = 30
