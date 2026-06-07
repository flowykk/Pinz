FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY shared /opt/shared
RUN pip install /opt/shared

COPY proto/gen /opt/proto_gen
ENV PYTHONPATH=/opt/proto_gen

COPY search_service/pyproject.toml /app/pyproject.toml
COPY search_service/src /app/src
RUN pip install /app

EXPOSE 50055
ENTRYPOINT ["python", "-m", "src.server"]
