# Единая точка генерации proto: Backend/proto -> auth-service/pkg/proto, api-gateway-service/pkg/proto
.PHONY: proto
proto:
	mkdir -p auth-service/pkg/proto api-gateway-service/pkg/proto
	protoc -I proto \
		--go_out=auth-service --go_opt=module=pinz/backend/auth-service \
		--go-grpc_out=auth-service --go-grpc_opt=module=pinz/backend/auth-service \
		proto/auth.proto
	protoc -I proto \
		--go_out=api-gateway-service --go_opt=module=pinz/backend/api-gateway-service --go_opt=Mauth.proto=pinz/backend/api-gateway-service/pkg/proto \
		--go-grpc_out=api-gateway-service 		--go-grpc_opt=module=pinz/backend/api-gateway-service --go-grpc_opt=Mauth.proto=pinz/backend/api-gateway-service/pkg/proto \
		proto/auth.proto

.PHONY: swagger
swagger:
	$(MAKE) -C api-gateway-service swagger

# Local linting
.PHONY: lint lint-api lint-auth check-lint
lint: lint-api lint-auth

lint-api:
	cd api-gateway-service && golangci-lint run ./...

lint-auth:
	cd auth-service && golangci-lint run ./...

# Infra: Postgres + Redis (Docker Compose, accessible via host.docker.internal from k8s)
.PHONY: infra-up infra-down
infra-up:
	docker compose -f docker-compose.infra.yml up -d

infra-down:
	docker compose -f docker-compose.infra.yml down

# Observability stack: OTel Collector, Tempo, Prometheus, Loki, Grafana — all in k8s
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

# Open Grafana via port-forward (runs in foreground, Ctrl+C to stop)
grafana:
	@echo "Grafana → http://localhost:3000"
	kubectl port-forward svc/grafana 3000:3000

# Direct port-forward to api-gateway (bypasses Lima conflicts on :80/:443)
# Access: http://localhost:8080
dev:
	@echo "API    → http://localhost:8080"
	@echo "Swagger → http://localhost:8080/swagger/index.html"
	kubectl port-forward svc/api-gateway 8080:8080

# k8s: build images inside Minikube, apply Istio, deploy via helmfile
.PHONY: k8s-build k8s-deploy k8s-istio
k8s-build:
	eval $$(minikube docker-env) && \
	docker build -t backend-auth-service:latest -f auth-service/Dockerfile . && \
	docker build -t backend-api-gateway-service:latest -f api-gateway-service/Dockerfile .

k8s-deploy:
	helmfile -f helmfile.local.yaml.gotmpl apply && \
	kubectl rollout restart deployment/api-gateway deployment/auth-service

k8s-istio:
	kubectl apply -f k8s-istio/

# Full local stack: infra (docker) + observability (k8s) + app (k8s)
.PHONY: all-up all-down
all-up: infra-up obs-up k8s-build k8s-istio k8s-deploy

all-down:
	kubectl delete -f k8s/k8s-observability.yaml || true
	helmfile -f helmfile.local.yaml.gotmpl destroy || true
	docker compose -f docker-compose.infra.yml down
