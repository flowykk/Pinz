# vLLM requires CUDA; use the official vLLM base image for GPU deployments.
# For CPU-only testing, swap to python:3.11-slim and install vllm[cpu].
FROM vllm/vllm-openai:v0.4.2

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir fastapi uvicorn[standard] pydantic-settings

COPY app/ ./app/

ENV MODEL_NAME=meta-llama/Meta-Llama-Guard-2-8B
ENV TENSOR_PARALLEL_SIZE=1
ENV MAX_MODEL_LEN=4096

EXPOSE 8004
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8004"]
