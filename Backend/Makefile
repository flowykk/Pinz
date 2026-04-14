GOPROXY ?= https://proxy.golang.org,direct
export GOPROXY

.PHONY: proto
proto:
	mkdir -p auth-service/pkg/proto api-gateway-service/pkg/proto trip-service/pkg/proto
	protoc -I proto \
		--go_out=auth-service --go_opt=module=pinz/backend/auth-service \
		--go-grpc_out=auth-service --go-grpc_opt=module=pinz/backend/auth-service \
		proto/auth.proto
	protoc -I proto \
		--go_out=api-gateway-service --go_opt=module=pinz/backend/api-gateway-service --go_opt=Mauth.proto=pinz/backend/api-gateway-service/pkg/proto \
		--go-grpc_out=api-gateway-service 		--go-grpc_opt=module=pinz/backend/api-gateway-service --go-grpc_opt=Mauth.proto=pinz/backend/api-gateway-service/pkg/proto \
		proto/auth.proto
	protoc -I proto \
		--go_out=trip-service --go_opt=module=pinz/backend/trip-service \
		--go-grpc_out=trip-service --go-grpc_opt=module=pinz/backend/trip-service \
		proto/trip.proto
	protoc -I proto \
		--go_out=api-gateway-service --go_opt=module=pinz/backend/api-gateway-service --go_opt=Mtrip.proto=pinz/backend/api-gateway-service/pkg/proto \
		--go-grpc_out=api-gateway-service --go-grpc_opt=module=pinz/backend/api-gateway-service --go-grpc_opt=Mtrip.proto=pinz/backend/api-gateway-service/pkg/proto \
		proto/trip.proto

.PHONY: mocks
mocks:
	cd auth-service && go generate ./internal/mocks/...
	cd api-gateway-service && go generate ./internal/mocks/...
	cd trip-service && go generate ./internal/mocks/...

.PHONY: swagger
swagger:
	$(MAKE) -C api-gateway-service swagger

SQLC_DOCKER_IMAGE ?= sqlc/sqlc:1.28.0
.PHONY: sqlc sqlc-check
sqlc:
	@if command -v sqlc >/dev/null 2>&1; then \
		(cd auth-service && sqlc generate) && (cd trip-service && sqlc generate); \
	elif command -v docker >/dev/null 2>&1; then \
		docker run --rm -v "$(PWD)/auth-service:/src" -w /src $(SQLC_DOCKER_IMAGE) generate && \
		docker run --rm -v "$(PWD)/trip-service:/src" -w /src $(SQLC_DOCKER_IMAGE) generate; \
	else \
		echo "Install sqlc (https://docs.sqlc.dev) or Docker to run $(SQLC_DOCKER_IMAGE)"; exit 1; \
	fi

sqlc-check: sqlc
	@if [ -n "$$(git -C .. status --porcelain -- Backend/auth-service/internal/db/sqlcdb Backend/auth-service/queries Backend/auth-service/sqlc.yaml Backend/trip-service/internal/db/sqlcdb Backend/trip-service/queries Backend/trip-service/sqlc.yaml 2>/dev/null)" ]; then \
		echo "sqlc drift: run 'make sqlc' from Backend and commit"; \
		git -C .. status --porcelain -- Backend/auth-service/internal/db/sqlcdb Backend/auth-service/queries Backend/trip-service/internal/db/sqlcdb Backend/trip-service/queries || true; \
		exit 1; \
	fi
	@echo "sqlc artifacts up to date ✓"

.PHONY: tidy
tidy:
	cd pkg/otel && go mod tidy
	cd auth-service && go mod tidy
	cd trip-service && go mod tidy
	cd api-gateway-service && go mod tidy
	cd e2e && go mod tidy

.PHONY: check-env
check-env:
	@bash scripts/check-env.sh

.PHONY: lint lint-api lint-auth lint-trip
lint: lint-api lint-auth lint-trip check-env

lint-api:
	cd api-gateway-service && golangci-lint run ./...

lint-auth:
	cd auth-service && golangci-lint run ./...

lint-trip:
	cd trip-service && golangci-lint run ./...

.PHONY: test test-unit test-unit-only test-integration test-e2e test-all
test: test-unit

DOCKER_HOST ?= $(shell [ -S /var/run/docker.sock ] || docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null)
export DOCKER_HOST
export TESTCONTAINERS_RYUK_DISABLED ?= true

test-unit:
	cd auth-service && go test -short ./...
	cd trip-service && go test -short ./...
	cd api-gateway-service && go test -short ./...

test-unit-only: test-unit

test-integration:
	cd auth-service && go test ./...
	cd trip-service && go test ./...
	cd api-gateway-service && go test ./...

test-e2e:
	cd e2e && PINZ_E2E_STRICT=1 go test ./...

test-all: test-unit test-e2e

.PHONY: coverage coverage-auth coverage-trip coverage-gateway
coverage: coverage-auth coverage-trip coverage-gateway

.PHONY: coverage-check
MIN_COVER ?= 80
coverage-check:
	@echo "Minimum coverage (services/handlers): $(MIN_COVER)%"
	@cd auth-service && go test -short ./internal/services/... -coverprofile=coverage.out
	@cd auth-service && go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$$3); print "auth-service internal/services:",$$3"%"; if ($$3+0 < $(MIN_COVER)+0) exit 1}'
	@cd trip-service && go test -short ./internal/services/... -coverprofile=coverage.out
	@cd trip-service && go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$$3); print "trip-service internal/services:",$$3"%"; if ($$3+0 < $(MIN_COVER)+0) exit 1}'
	@cd api-gateway-service && go test -short ./internal/handlers/... -coverprofile=coverage.out
	@cd api-gateway-service && go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$$3); print "api-gateway-service internal/handlers:",$$3"%"; if ($$3+0 < $(MIN_COVER)+0) exit 1}'

coverage-auth:
	cd auth-service && go test -short ./internal/services/... -coverprofile=coverage.out
	cd auth-service && go tool cover -func=coverage.out | awk '/^total:/ {print $$0}'

coverage-trip:
	cd trip-service && go test -short ./internal/services/... -coverprofile=coverage.out
	cd trip-service && go tool cover -func=coverage.out | awk '/^total:/ {print $$0}'

coverage-gateway:
	cd api-gateway-service && go test -short ./internal/handlers/... -coverprofile=coverage.out
	cd api-gateway-service && go tool cover -func=coverage.out | awk '/^total:/ {print $$0}'

.PHONY: infra-up infra-down
infra-up:
	docker compose -f docker-compose.infra.yml up -d

infra-down:
	docker compose -f docker-compose.infra.yml down

.PHONY: obs-up obs-down obs-status grafana
obs-up:
	kubectl apply -f k8s/k8s-observability.yaml
	@echo ""
	@echo "  Grafana access:  make grafana  (port-forward 3000)"
	@echo "  Check status:    make obs-status"
	@echo ""

obs-down:
	kubectl delete -f k8s/k8s-observability.yaml

obs-status:
	kubectl get pods -l component=observability

grafana:
	@echo "Grafana → http://localhost:3000"
	kubectl port-forward svc/grafana 3000:3000

dev:
	@echo "API    → http://localhost:8080"
	@echo "Swagger → http://localhost:8080/swagger/index.html"
	kubectl port-forward svc/api-gateway 8080:8080

.PHONY: k8s-build k8s-deploy k8s-istio
k8s-build:
	eval $$(minikube docker-env) && \
	docker build -t backend-auth-service:latest -f auth-service/Dockerfile . && \
	docker build -t backend-trip-service:latest -f trip-service/Dockerfile . && \
	docker build -t backend-api-gateway-service:latest -f api-gateway-service/Dockerfile .

k8s-deploy:
	set -a && [ -f .env ] && . ./.env; set +a && \
	helmfile -f helmfile.local.yaml.gotmpl apply && \
	kubectl rollout restart deployment/api-gateway deployment/auth-service deployment/trip-service

k8s-istio:
	kubectl apply -f k8s-istio/

.PHONY: all-up all-down
all-up: infra-up obs-up k8s-build k8s-istio k8s-deploy

all-down:
	kubectl delete -f k8s/k8s-observability.yaml || true
	helmfile -f helmfile.local.yaml.gotmpl destroy || true
	docker compose -f docker-compose.infra.yml down
