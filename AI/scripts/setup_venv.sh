#!/usr/bin/env bash
# Создаёт venv с Python 3.11 и устанавливает все зависимости AI Module.
# Запускать из корня AI/.
#
#   bash scripts/setup_venv.sh
#
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PYTHON311="/opt/homebrew/bin/python3.11"
if [[ ! -x "$PYTHON311" ]]; then
  # Пробуем стандартные пути
  for p in /usr/local/bin/python3.11 $(which python3.11 2>/dev/null); do
    [[ -x "$p" ]] && PYTHON311="$p" && break
  done
fi

if [[ ! -x "$PYTHON311" ]]; then
  echo "ERROR: Python 3.11 not found. Install: brew install python@3.11" >&2
  exit 1
fi

echo "Using Python: $PYTHON311 ($($PYTHON311 --version))"

# --- Создаём venv ---
if [[ -d .venv ]]; then
  echo "venv already exists at .venv, skipping creation."
else
  $PYTHON311 -m venv .venv
  echo "Created .venv"
fi

PIP=".venv/bin/pip"

# --- Upgrade pip ---
$PIP install --upgrade pip wheel setuptools

# --- Генерируем proto-стабы ---
echo ""
echo "=== Generating proto stubs ==="
$PIP install grpcio-tools --quiet
bash scripts/gen_proto.sh

echo ""
echo "=== Installing shared library ==="
$PIP install -e shared/

echo ""
echo "=== Installing core dependencies ==="
$PIP install torch --index-url https://download.pytorch.org/whl/cpu
$PIP install transformers sentencepiece pillow numpy scipy httpx pydantic pydantic-settings structlog

echo ""
echo "=== Installing gRPC ==="
# grpcio и grpcio-tools должны быть одной версии — иначе сгенерированные стабы несовместимы
$PIP install "grpcio==1.81.0" "grpcio-tools==1.81.0" "grpcio-health-checking==1.81.0" protobuf

echo ""
echo "=== Installing storage clients ==="
$PIP install qdrant-client "meilisearch-python-sdk>=4.0"

echo ""
echo "=== Installing NATS client ==="
$PIP install "nats-py>=2.7"

echo ""
echo "=== Installing Triton client ==="
# Только grpc transport; gpu не нужен
$PIP install "tritonclient[grpc]" || echo "WARNING: tritonclient failed — Triton-stub will be used for tests"

echo ""
echo "=== Installing service packages (editable) ==="
for svc in ml_orchestrator moderation_service embedding_service classifier_service similarity_service search_service; do
  echo "  → $svc"
  $PIP install -e "${svc}/" --no-deps --quiet
done

echo ""
echo "=== Installing dev/test tools ==="
$PIP install pytest pytest-asyncio pytest-timeout respx coverage ruff

echo ""
echo "========================================================"
echo "Setup complete! Activate with:"
echo "  source .venv/bin/activate"
echo ""
echo "Next steps:"
echo "  1. make infra-up                    # NATS + Qdrant + Meilisearch"
echo "  2. bash scripts/run_local.sh        # все 6 сервисов"
echo "  3. pytest tests/unit/               # unit-тесты (без инфры)"
echo "  4. pytest tests/integration/        # e2e (требует запущенных сервисов)"
echo "========================================================"
