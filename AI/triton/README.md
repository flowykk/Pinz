# Triton Inference Server

В `model_repository/` лежат конфиги моделей. Сами веса (`*.onnx`) генерируются
скриптом `scripts/export_models.py` и не коммитятся в git.

## Подготовка моделей

```bash
# Из корня AI/
pip install torch transformers onnx
python triton/scripts/export_models.py
```

Скрипт скачивает:
- `google/siglip-base-patch16-224` — vision + text branches
- `Falconsai/nsfw_image_detection` — ViT для NSFW

и кладёт ONNX-веса в нужные подпапки.

## Запуск

```bash
docker run --rm -p 8000:8000 -p 8001:8001 -p 8002:8002 \
  -v $(pwd)/triton/model_repository:/models \
  nvcr.io/nvidia/tritonserver:24.05-py3 \
  tritonserver --model-repository=/models --strict-model-config=false
```

В docker-compose сервис называется `triton`.
