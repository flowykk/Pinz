#!/usr/bin/env bash
# Запускать параллельно с k6/scenarios/resilience.js.
# Поочерёдно гасит стабы и поднимает обратно: проверяем graceful degradation.
set -euo pipefail
NS=pinz-loadtest

down () { kubectl -n "$NS" scale deploy "$1" --replicas=0; }
up   () { kubectl -n "$NS" scale deploy "$1" --replicas=1; }

echo "minutes 5..7: minio off"
sleep 300; down minio
sleep 120; up   minio

echo "minutes 10..12: mailpit off"
sleep 180; down mailpit
sleep 120; up   mailpit

echo "minutes 15..17: geo-stub off"
sleep 180; down geo-stub
sleep 120; up   geo-stub

echo "done"
