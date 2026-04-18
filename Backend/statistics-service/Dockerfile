# Build context must be Backend/ (the repo root for Go modules).
# docker build -f statistics-service/Dockerfile .
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Download deps for the shared pkg/otel module first (better layer caching).
COPY pkg/otel/go.mod pkg/otel/go.sum ./pkg/otel/
RUN cd pkg/otel && go mod download

# Download deps for the service itself.
COPY statistics-service/go.mod statistics-service/go.sum ./statistics-service/
RUN cd statistics-service && go mod download

# Copy all sources and build.
COPY pkg/otel/ ./pkg/otel/
COPY statistics-service/ ./statistics-service/
RUN cd statistics-service && CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/statistics-service ./cmd/main.go

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/bin/statistics-service /app/statistics-service
EXPOSE 50053
ENTRYPOINT ["/app/statistics-service"]
