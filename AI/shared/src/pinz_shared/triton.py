"""Утилиты для общения с Triton Inference Server по gRPC."""

from __future__ import annotations

import numpy as np
import tritonclient.grpc.aio as triton_aio


class TritonAsyncClient:
    def __init__(self, url: str) -> None:
        self._url = url
        self._client: triton_aio.InferenceServerClient | None = None

    async def connect(self) -> None:
        self._client = triton_aio.InferenceServerClient(url=self._url, verbose=False)

    async def close(self) -> None:
        if self._client is not None:
            await self._client.close()

    @property
    def client(self) -> triton_aio.InferenceServerClient:
        if self._client is None:
            raise RuntimeError("TritonAsyncClient is not connected")
        return self._client

    async def infer(
        self,
        model: str,
        inputs: dict[str, np.ndarray],
        outputs: list[str],
        *,
        model_version: str = "",
    ) -> dict[str, np.ndarray]:
        ins = []
        for name, arr in inputs.items():
            dtype = _np_to_triton_dtype(arr.dtype)
            t = triton_aio.InferInput(name, list(arr.shape), dtype)
            t.set_data_from_numpy(arr)
            ins.append(t)

        outs = [triton_aio.InferRequestedOutput(n) for n in outputs]
        resp = await self.client.infer(
            model_name=model,
            inputs=ins,
            outputs=outs,
            model_version=model_version,
        )
        return {n: resp.as_numpy(n) for n in outputs}


def _np_to_triton_dtype(dt: np.dtype) -> str:
    mapping = {
        np.dtype("float32"): "FP32",
        np.dtype("float16"): "FP16",
        np.dtype("int32"): "INT32",
        np.dtype("int64"): "INT64",
        np.dtype("uint8"): "UINT8",
        np.dtype("bool"): "BOOL",
    }
    if dt not in mapping:
        raise TypeError(f"unsupported numpy dtype for triton: {dt}")
    return mapping[dt]
