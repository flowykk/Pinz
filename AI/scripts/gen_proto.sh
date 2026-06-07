#!/usr/bin/env bash
# Генерирует Python-стабы для всех .proto в proto/ в proto/gen/pinz_ai_proto/.
# Запускается из корня AI/. Требует grpcio-tools (есть в shared/pyproject.toml).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROTO_DIR="$ROOT/proto"
OUT_DIR="$PROTO_DIR/gen/pinz_ai_proto"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"
touch "$OUT_DIR/__init__.py"

# Используем python из venv если есть, иначе системный
PYTHON="${ROOT}/.venv/bin/python"
if [[ ! -x "$PYTHON" ]]; then
  PYTHON="$(command -v python3.11 2>/dev/null || command -v python3 || command -v python)"
fi
echo "Using Python: $PYTHON"

"$PYTHON" -m grpc_tools.protoc \
  --proto_path="$PROTO_DIR" \
  --python_out="$OUT_DIR" \
  --grpc_python_out="$OUT_DIR" \
  --pyi_out="$OUT_DIR" \
  "$PROTO_DIR"/*.proto

# grpc_tools генерирует абсолютные импорты "import common_pb2" — починим.
"$PYTHON" - <<PY
import pathlib, re
out = pathlib.Path("$OUT_DIR")
for f in out.glob("*_pb2*.py"):
    s = f.read_text()
    s = re.sub(r"^import (common|moderation|embedding|classifier|similarity|search)_pb2",
               r"from . import \1_pb2", s, flags=re.MULTILINE)
    s = re.sub(r"^import (common|moderation|embedding|classifier|similarity|search)_pb2_grpc",
               r"from . import \1_pb2_grpc", s, flags=re.MULTILINE)
    f.write_text(s)
print(f"proto stubs generated in {out}")
PY
