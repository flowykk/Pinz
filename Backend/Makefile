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

# Local k8s deployment (Minikube + Istio)
.PHONY: infra-up infra-down k8s-build k8s-deploy k8s-istio all-up
infra-up:
	docker-compose -f docker-compose.infra.yml up -d

infra-down:
	docker-compose -f docker-compose.infra.yml down

k8s-build:
	eval $$(minikube docker-env) && \
	docker build -t pinz-auth-service:latest -f auth-service/Dockerfile auth-service && \
	docker build -t pinz-api-gateway:latest -f api-gateway-service/Dockerfile api-gateway-service

k8s-deploy:
	helmfile -f helmfile.local.yaml.gotmpl apply \
	kubectl rollout restart deployment/api-gateway deployment/auth-service

k8s-istio:
	kubectl apply -f k8s-istio/

# Full deployment: infra + build + istio + deploy
.PHONY: all-up
all-up: infra-up k8s-build k8s-istio k8s-deploy
