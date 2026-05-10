"""Singleton wrapper around vLLM AsyncLLMEngine."""
from __future__ import annotations

from vllm import AsyncEngineArgs, AsyncLLMEngine
from vllm.sampling_params import SamplingParams

from app.config import settings

_engine: AsyncLLMEngine | None = None


def get_engine() -> AsyncLLMEngine:
    global _engine
    if _engine is None:
        args = AsyncEngineArgs(
            model=settings.model_name,
            tensor_parallel_size=settings.tensor_parallel_size,
            max_model_len=settings.max_model_len,
            gpu_memory_utilization=settings.gpu_memory_utilization,
            trust_remote_code=True,
        )
        _engine = AsyncLLMEngine.from_engine_args(args)
    return _engine


def build_sampling_params() -> SamplingParams:
    return SamplingParams(
        temperature=settings.temperature,
        max_tokens=settings.max_new_tokens,
    )
