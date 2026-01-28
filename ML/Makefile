## ML module convenience targets.
##
## Notes:
## - Run these targets from repo root: `make -f ML/Makefile <target>`
## - We `cd ML` so Python imports are stable and local paths work.

.PHONY: install notebooks proto run-moderation run-trip-classifier run-search run-gateway-http run-gateway-grpc

install:
	cd ML && poetry install

notebooks:
	cd ML && poetry run jupyter lab notebooks

## Generate gRPC Python code from proto files.
proto:
	cd ML/services/api_gateway_service && \
	python -m grpc_tools.protoc -I./proto \
		--python_out=./proto_gen \
		--grpc_python_out=./proto_gen \
		./proto/moderation.proto

run-moderation:
	cd ML && poetry run uvicorn services.moderation_service.main:app --reload --port 8001

run-trip-classifier:
	cd ML && poetry run uvicorn services.trip_classifier_service.main:app --reload --port 8002

run-search:
	cd ML && poetry run uvicorn services.search_service.main:app --reload --port 8003

## HTTP gateway (optional).
run-gateway-http:
	cd ML && poetry run uvicorn services.api_gateway_service.main:app --reload --port 8000

## gRPC gateway (ModerationService) defined in proto.
run-gateway-grpc:
	cd ML/services/api_gateway_service && python grpc_server.py

## Docker Compose helpers for local multi-service runs
compose-up:
	cd ML && docker-compose up --build -d

compose-down:
	cd ML && docker-compose down
