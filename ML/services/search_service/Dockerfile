FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ ./app/

ENV ELASTICSEARCH_URL=http://elasticsearch:9200
ENV QDRANT_URL=http://qdrant:6333
ENV EMBEDDING_SERVICE_URL=http://embedding-service:8002

EXPOSE 8003
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8003"]
