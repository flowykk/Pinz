FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ ./app/

ENV MODEL_CACHE_DIR=/model_cache
ENV TEXT_EMBED_MODEL=intfloat/multilingual-e5-large
ENV IMAGE_EMBED_MODEL=ViT-L-14
ENV IMAGE_EMBED_PRETRAINED=openai

EXPOSE 8002
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8002"]
