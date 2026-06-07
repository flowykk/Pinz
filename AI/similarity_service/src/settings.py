from pinz_shared import BaseServiceSettings


class Settings(BaseServiceSettings):
    service_name: str = "similarity-service"
    grpc_port: int = 50054
