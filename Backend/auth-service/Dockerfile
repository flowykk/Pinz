# Build context must be Backend/ (the repo root for Go modules).
# docker build -f auth-service/Dockerfile .
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Download deps for the shared pkg/otel module first (better layer caching).
COPY pkg/otel/go.mod pkg/otel/go.sum ./pkg/otel/
RUN cd pkg/otel && go mod download

# Download deps for the service itself.
COPY auth-service/go.mod auth-service/go.sum ./auth-service/
RUN cd auth-service && go mod download

# Copy all sources and build.
COPY pkg/otel/ ./pkg/otel/
COPY auth-service/ ./auth-service/
RUN cd auth-service && CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/auth-service ./cmd/main.go

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/bin/auth-service /app/auth-service
EXPOSE 50051
ENTRYPOINT ["/app/auth-service"]
