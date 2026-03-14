# Build context must be Backend/ (the repo root for Go modules).
# docker build -f trip-service/Dockerfile .
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Download deps for the shared pkg/otel module first (better layer caching).
COPY pkg/otel/go.mod pkg/otel/go.sum ./pkg/otel/
RUN cd pkg/otel && go mod download

# Download deps for the service itself.
COPY trip-service/go.mod trip-service/go.sum ./trip-service/
RUN cd trip-service && go mod download

# Copy all sources and build.
COPY pkg/otel/ ./pkg/otel/
COPY trip-service/ ./trip-service/
RUN cd trip-service && CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/trip-service ./cmd/main.go

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/bin/trip-service /app/trip-service
EXPOSE 50052
ENTRYPOINT ["/app/trip-service"]
