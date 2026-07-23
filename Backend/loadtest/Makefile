.PHONY: help seed smoke baseline load soak stress resilience cleanup

help:
	@echo "Targets:"
	@echo "  seed         — наполнить БД 10000 тест-юзеров и собрать credentials.json"
	@echo "  smoke        — 1 VU, 1 мин, все journeys по разу"
	@echo "  baseline     — 50 VU, 10 мин, mix"
	@echo "  load         — ramp 0→1000 VU за 5 мин, hold 30 мин (см. scenarios/load.js)"
	@echo "  soak         — 300 VU, 4 ч (запускать через nohup, ночью)"
	@echo "  stress       — ramp до отказа (только с runner-VPS)"
	@echo "  resilience   — 200 VU + параллельный resilience-faults.sh"
	@echo "  cleanup      — снести test-users из auth-БД и loadtest/ префикс из S3"
	@echo ""
	@echo "Required env: BASE_URL, LOADTEST_DB_URL (для seed/cleanup), S3_* (для cleanup)"

BASE_URL ?= $(error set BASE_URL, e.g. http://<VPS_IP>:8080)
USERS    ?= 10000
CONC     ?= 32

seed:
	cd seed && go run . -base-url=$(BASE_URL) -db-url=$(LOADTEST_DB_URL) -users=$(USERS) -concurrency=$(CONC) -out=credentials.json

smoke:
	k6 run -e BASE_URL=$(BASE_URL) -e CREDENTIALS_PATH=./seed/credentials.json k6/scenarios/smoke.js

baseline:
	k6 run -e BASE_URL=$(BASE_URL) -e CREDENTIALS_PATH=./seed/credentials.json --out json=reports/baseline.json k6/scenarios/baseline.js

load:
	k6 run -e BASE_URL=$(BASE_URL) -e CREDENTIALS_PATH=./seed/credentials.json --out json=reports/load.json k6/scenarios/load.js

soak:
	nohup k6 run -e BASE_URL=$(BASE_URL) -e CREDENTIALS_PATH=./seed/credentials.json --out json=reports/soak.json k6/scenarios/soak.js > reports/soak.log 2>&1 &
	@echo "soak running in background; tail -f reports/soak.log"

stress:
	k6 run -e BASE_URL=$(BASE_URL) -e CREDENTIALS_PATH=./seed/credentials.json --out json=reports/stress.json k6/scenarios/stress.js

resilience:
	@echo "Запусти параллельно: bash scripts/resilience-faults.sh"
	k6 run -e BASE_URL=$(BASE_URL) -e CREDENTIALS_PATH=./seed/credentials.json --out json=reports/resilience.json k6/scenarios/resilience.js

cleanup:
	@test -n "$(LOADTEST_DB_URL)" || (echo "LOADTEST_DB_URL is required"; exit 1)
	psql "$(LOADTEST_DB_URL)" -f scripts/cleanup.sql
	@test -n "$(S3_BUCKET)" -a -n "$(S3_ENDPOINT)" || (echo "S3_BUCKET/S3_ENDPOINT not set, пропускаю S3 cleanup"; exit 0)
	S3_BUCKET=$(S3_BUCKET) S3_ENDPOINT=$(S3_ENDPOINT) S3_KEY_PREFIX=$(S3_KEY_PREFIX) bash scripts/cleanup-s3.sh
