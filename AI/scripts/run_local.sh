#!/usr/bin/env bash
# Запускает все 6 AI микросервисов нативно в venv.
# Каждый сервис — в своём фоновом процессе.
# PID'ы пишутся в /tmp/pinz-ai-pids для последующего kill.
#
# Использование:
#   bash scripts/run_local.sh          # старт
#   bash scripts/run_local.sh stop     # остановка

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VENV="$ROOT/.venv"
ENV_FILE="$ROOT/.env.local"
PID_FILE="/tmp/pinz-ai-pids"

if [[ "${1:-}" == "stop" ]]; then
  if [[ -f "$PID_FILE" ]]; then
    echo "Stopping Pinz AI services..."
    while IFS= read -r pid; do
      kill "$pid" 2>/dev/null && echo "  killed $pid" || true
    done < "$PID_FILE"
    rm -f "$PID_FILE"
    echo "Done."
  else
    echo "No PID file found at $PID_FILE."
  fi
  exit 0
fi

# --- Проверки ---
if [[ ! -d "$VENV" ]]; then
  echo "ERROR: venv not found at $VENV. Run: bash scripts/setup_venv.sh" >&2
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: $ENV_FILE not found. Copy from .env.local (it should already be there)." >&2
  exit 1
fi

export $(grep -v '^\s*#' "$ENV_FILE" | grep -v '^\s*$' | xargs)

# --- PYTHONPATH ---
# Каждый сервис запускается из своей директории как «python -m src.server»
# (так же, как в Dockerfile: WORKDIR=/app/<svc> + ENTRYPOINT python -m src.server).
# PYTHONPATH нужен только для shared-библиотеки и proto-стабов.
PROTO_GEN="$ROOT/proto/gen"
SHARED_SRC="$ROOT/shared/src"
export PYTHONPATH="$PROTO_GEN:$SHARED_SRC"

PYTHON="$VENV/bin/python"
LOG_DIR="$ROOT/logs"
mkdir -p "$LOG_DIR"
> "$PID_FILE"

start_svc() {
  local name="$1"
  local module="$2"       # e.g. src.server  или  src.main
  local svc_dir="$ROOT/$name"
  local log="$LOG_DIR/${name}.log"
  echo "Starting $name → $log"
  # cd в директорию сервиса — имитирует WORKDIR из Dockerfile
  (cd "$svc_dir" && exec $PYTHON -m "$module") >> "$log" 2>&1 &
  echo $! >> "$PID_FILE"
  echo "  PID=$!"
}

# Порядок важен: сначала downstream, потом orchestrator
start_svc "moderation_service"  "src.server"    # port 50051
start_svc "embedding_service"   "src.server"    # port 50052
start_svc "classifier_service"  "src.server"    # port 50053
start_svc "similarity_service"  "src.server"    # port 50054
start_svc "search_service"      "src.server"    # port 50055

# Даём downstream сервисам 3 секунды на старт
sleep 3

start_svc "ml_orchestrator"     "src.main"      # NATS consumer

echo ""
echo "All services started. Logs in $LOG_DIR/"
echo "Stop with: bash scripts/run_local.sh stop"
echo ""
echo "Tail orchestrator log:"
echo "  tail -f $LOG_DIR/ml_orchestrator.log"
