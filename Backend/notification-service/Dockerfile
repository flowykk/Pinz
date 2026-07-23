# Build context must be Backend/ (the repo root for Go modules).
# docker build -f notification-service/Dockerfile .
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY pkg/otel/go.mod pkg/otel/go.sum ./pkg/otel/
RUN cd pkg/otel && go mod download

COPY notification-service/go.mod notification-service/go.sum ./notification-service/
RUN cd notification-service && go mod download

COPY pkg/otel/ ./pkg/otel/
COPY notification-service/ ./notification-service/
RUN cd notification-service && CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/notification-service ./cmd/main.go

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/bin/notification-service /app/notification-service
EXPOSE 50054
ENTRYPOINT ["/app/notification-service"]
