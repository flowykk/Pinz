# Pinz Load Testing

Инфраструктура нагрузочного тестирования backend Pinz. Стратегия и обоснование — `vkr/loadTestingStrategy.md`.

> ВАЖНО: запускать только на отдельном loadtest-VPS. На прод (`pinz.website`) ничего не выкатывать. Все стабы — `loadtest/manifests/README.md`.

## Структура

```
loadtest/
  k6/                  # сценарии и хелперы (smoke/baseline/load/soak/stress/resilience)
  seed/                # Go-утилита: создаёт тест-юзеров, делает dev-login, пишет credentials.json
  manifests/           # MinIO + Mailpit + geo-stub в namespace pinz-loadtest
  scripts/             # cleanup.sql, cleanup-s3.sh, resilience-faults.sh, grafana-dashboard.json
  config/              # env.loadtest.example
  reports/             # артефакты прогонов (gitignored)
  Makefile             # цели seed/smoke/baseline/load/soak/stress/resilience/cleanup
```

## Однократная подготовка стенда

1. Берём временный VPS (Hetzner CPX31 / Selectel / Timeweb): 4 vCPU / 8 ГБ / 80 ГБ.
2. На VPS:
   ```bash
   git clone <repo> && cd Pinz/Backend
   ./setup-server.sh                       # docker, k3s, helm, helmfile, istio
   docker compose -f docker-compose.infra.yml up -d   # 4 PostgreSQL + Redis на хосте
   kubectl apply -f loadtest/manifests/minio.yaml
   kubectl apply -f loadtest/manifests/mailpit.yaml
   docker build -t pinz/geo-stub:loadtest loadtest/manifests/geo-stub/
   kubectl apply -f loadtest/manifests/geo-stub/k8s.yaml
   helmfile -f helmfile.loadtest.yaml.gotmpl sync
   ```
3. Проверяем: `curl http://<VPS_IP>:8080/health` (через NodePort/Istio Gateway).

## Прогон с ноута

```bash
brew install k6 go awscli
ulimit -n 65536

cd Backend/loadtest
cp config/env.loadtest.example .env
# Отредактировать .env: BASE_URL, LOADTEST_DB_URL, S3_*

source .env

# 1) Сидим тест-данные (один раз)
make seed BASE_URL=$BASE_URL USERS=10000

# 2) Smoke
make smoke BASE_URL=$BASE_URL

# 3) Baseline
make baseline BASE_URL=$BASE_URL

# 4) Целевой Load
make load BASE_URL=$BASE_URL

# 5) Soak (на ночь)
make soak BASE_URL=$BASE_URL
```

## Grafana

Импорт дашборда: `loadtest/scripts/grafana-dashboard.json` → Grafana → Dashboards → Import. Туннель с ноута:
```bash
ssh -L 3000:localhost:3000 root@<VPS_IP>
```

## Чистка после прогона

```bash
make cleanup LOADTEST_DB_URL=$LOADTEST_DB_URL S3_ENDPOINT=$S3_ENDPOINT S3_BUCKET=$S3_BUCKET S3_KEY_PREFIX=loadtest/
kubectl delete namespace pinz-loadtest
# выключить VPS у провайдера
```

## SLO

Зафиксированы в `k6/lib/slo.js` и автоматически проверяются k6. Смотрите таблицу классов ручек в `vkr/loadTestingStrategy.md` §4.

## Отладка

- **`dropped_iterations > 0`** в выводе k6 — bottleneck-ом стал ноутбук или интернет. Перезапустить с runner-VPS в том же ДЦ.
- **5xx на 1000 VU** — открыть Grafana, искать всплеск CPU/RAM/connections.
- **WS события не приходят** — проверить, что в trip-service worker запущен (`kubectl -n pinz-loadtest logs deploy/trip-service | grep "ML task consumer"`).

## Безопасность

- `AUTH_DEV_LOGIN_ENABLED=true` и `DEV_LOGIN_PROXY_ENABLED=true` — **только** в `helmfile.loadtest.yaml.gotmpl`. На проде значения по умолчанию `false`.
- `S3_KEY_PREFIX=loadtest/` гарантирует, что прод-бакет (если используется) не засоряется тест-объектами.
- Тест-пользователи помечены `users.is_test = true` (миграция 00003) и сносятся одной командой `make cleanup`.
