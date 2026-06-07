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

COPY embedding_service/pyproject.toml /app/pyproject.toml
COPY embedding_service/src /app/src
RUN pip install /app

# Toкенизатор SigLIP (нужен только tokenizer.json, не модель)
ARG SIGLIP_MODEL_ID=google/siglip-base-patch16-224
RUN python -c "from transformers import AutoTokenizer; AutoTokenizer.from_pretrained('$SIGLIP_MODEL_ID')"

EXPOSE 50052
ENTRYPOINT ["python", "-m", "src.server"]
