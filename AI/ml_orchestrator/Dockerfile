FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1

WORKDIR /app

# system deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Сначала ставим shared
COPY shared /opt/shared
RUN pip install /opt/shared

# Сгенерированные proto-стабы
COPY proto/gen /opt/proto_gen
ENV PYTHONPATH=/opt/proto_gen

# Сам сервис
COPY ml_orchestrator/pyproject.toml /app/pyproject.toml
COPY ml_orchestrator/src /app/src
RUN pip install /app

ENTRYPOINT ["python", "-m", "src.main"]
