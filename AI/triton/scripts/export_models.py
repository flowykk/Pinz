"""Экспорт HuggingFace моделей в ONNX и копирование в triton/model_repository.

Запуск:
    python triton/scripts/export_models.py

Создаёт:
    triton/model_repository/siglip_vision/1/model.onnx  
    triton/model_repository/siglip_text/1/model.onnx    
    triton/model_repository/nsfw_detector/1/model.onnx  
"""

from __future__ import annotations

import argparse
from pathlib import Path

import torch
from transformers import AutoImageProcessor, AutoModelForImageClassification, AutoModel, AutoTokenizer


ROOT = Path(__file__).resolve().parents[1] / "model_repository"


def _export(model, dummy, path, input_names, output_names, dynamic_axes):
    """Экспорт в ONNX"""
    kwargs = dict(
        input_names=input_names,
        output_names=output_names,
        dynamic_axes=dynamic_axes,
        opset_version=17,
    )
    
    try:
        with torch.no_grad():
            torch.onnx.export(model, dummy, path, dynamo=False, **kwargs)
    except TypeError:
        with torch.no_grad():
            torch.onnx.export(model, dummy, path, **kwargs)


def export_siglip(model_id: str = "google/siglip-base-patch16-224") -> None:
    model = AutoModel.from_pretrained(model_id).eval()

    # ----- Vision branch: output [batch, 768] -----
    class Vision(torch.nn.Module):
        def __init__(self, m):
            super().__init__()
            self.m = m

        def forward(self, pixel_values):
            out = self.m.get_image_features(pixel_values=pixel_values)
            if out.ndim == 3:
                out = out.mean(dim=1)  
            return out 

    out_v = ROOT / "siglip_vision" / "1" / "model.onnx"
    out_v.parent.mkdir(parents=True, exist_ok=True)
    dummy_img = torch.randn(2, 3, 224, 224)
    _export(
        Vision(model), (dummy_img,), out_v.as_posix(),
        input_names=["pixel_values"],
        output_names=["image_embeds"],
        dynamic_axes={"pixel_values": {0: "batch"}, "image_embeds": {0: "batch"}},
    )
    _verify(out_v, expected_ndim=2, expected_last_dim=768)
    print(f"[ok] {out_v}")

    # ----- Text branch: -----
    class Text(torch.nn.Module):
        def __init__(self, m):
            super().__init__()
            self.m = m

        def forward(self, input_ids):
            out = self.m.get_text_features(input_ids=input_ids)
            if out.ndim == 3:
                out = out.mean(dim=1)  
            return out 

    out_t = ROOT / "siglip_text" / "1" / "model.onnx"
    out_t.parent.mkdir(parents=True, exist_ok=True)
    dummy_ids = torch.zeros(2, 64, dtype=torch.long)
    _export(
        Text(model), (dummy_ids,), out_t.as_posix(),
        input_names=["input_ids"],
        output_names=["text_embeds"],
        dynamic_axes={"input_ids": {0: "batch"}, "text_embeds": {0: "batch"}},
    )
    _verify(out_t, expected_ndim=2, expected_last_dim=768)
    print(f"[ok] {out_t}")


def export_nsfw(model_id: str = "Falconsai/nsfw_image_detection") -> None:
    model = AutoModelForImageClassification.from_pretrained(model_id).eval()

    class NSFW(torch.nn.Module):
        def __init__(self, m):
            super().__init__()
            self.m = m

        def forward(self, pixel_values):
            return self.m(pixel_values=pixel_values).logits 

    out = ROOT / "nsfw_detector" / "1" / "model.onnx"
    out.parent.mkdir(parents=True, exist_ok=True)
    dummy = torch.randn(2, 3, 224, 224)
    _export(
        NSFW(model), (dummy,), out.as_posix(),
        input_names=["pixel_values"],
        output_names=["logits"],
        dynamic_axes={"pixel_values": {0: "batch"}, "logits": {0: "batch"}},
    )
    _verify(out, expected_ndim=2, expected_last_dim=2)
    print(f"[ok] {out}")


def _verify(path: Path, expected_ndim: int, expected_last_dim: int) -> None:
    """Проверить что ONNX-модель имеет правильную форму вывода."""
    try:
        import onnxruntime as ort
        sess = ort.InferenceSession(str(path), providers=["CPUExecutionProvider"])
        shape = sess.get_outputs()[0].shape
        assert len(shape) == expected_ndim, f"expected {expected_ndim}D got {len(shape)}D: {shape}"
        assert shape[-1] == expected_last_dim, f"last dim expected {expected_last_dim}, got {shape}"
        print(f"    verified shape={shape}")
    except ImportError:
        print("    (onnxruntime not installed, skipping shape verification)")


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--skip-siglip", action="store_true")
    p.add_argument("--skip-nsfw", action="store_true")
    args = p.parse_args()

    if not args.skip_siglip:
        export_siglip()
    if not args.skip_nsfw:
        export_nsfw()


if __name__ == "__main__":
    main()
