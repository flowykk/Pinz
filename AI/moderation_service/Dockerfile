FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1 HF_HOME=/models

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY shared /opt/shared
RUN pip install /opt/shared

COPY proto/gen /opt/proto_gen
ENV PYTHONPATH=/opt/proto_gen

COPY moderation_service/pyproject.toml /app/pyproject.toml
COPY moderation_service/src /app/src
RUN pip install /app

# Скачаем модель токсичности в образ, чтобы не дёргать сеть при старте
ARG TOXIC_MODEL_ID=textdetox/xlmr-large-toxicity-classifier
RUN python -c "from transformers import AutoTokenizer, AutoModelForSequenceClassification; \
    AutoTokenizer.from_pretrained('$TOXIC_MODEL_ID'); \
    AutoModelForSequenceClassification.from_pretrained('$TOXIC_MODEL_ID')"

EXPOSE 50051
ENTRYPOINT ["python", "-m", "src.server"]
