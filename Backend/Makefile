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
		--go-grpc_out=api-gateway-service --go-grpc_opt=module=pinz/backend/api-gateway-service --go-grpc_opt=Mauth.proto=pinz/backend/api-gateway-service/pkg/proto \
		proto/auth.proto
