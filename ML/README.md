ML Module for Pinz

This folder contains experimental notebooks, model training code, and microservice scaffolds for content moderation and related ML functionality.

Place dataset parquet files under `ML/data/` (e.g. `text_dataset.parquet`, `images_dataset.parquet`).

Quick start:

- Install Python dependencies via Poetry: `poetry install`
- Open notebooks in `ML/notebooks/` and run cells.
- Services are in `ML/services/` — each has `requirements.txt` and a `Dockerfile`.
