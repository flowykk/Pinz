# Build context must be Backend/ (the repo root for Go modules).
# docker build -f api-gateway-service/Dockerfile .
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Download deps for the shared pkg/otel module first (better layer caching).
COPY pkg/otel/go.mod pkg/otel/go.sum ./pkg/otel/
RUN cd pkg/otel && go mod download

# Download deps for the service itself.
COPY api-gateway-service/go.mod api-gateway-service/go.sum ./api-gateway-service/
RUN cd api-gateway-service && go mod download

# Copy all sources and build.
COPY pkg/otel/ ./pkg/otel/
COPY api-gateway-service/ ./api-gateway-service/
RUN cd api-gateway-service && CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/api-gateway-service ./cmd/main.go

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/bin/api-gateway-service /app/api-gateway-service
EXPOSE 8080
ENTRYPOINT ["/app/api-gateway-service"]
