FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o auth-service ./cmd/main.go

FROM scratch
COPY --from=builder /app/auth-service /app/auth-service
EXPOSE 50051
ENTRYPOINT ["/app/auth-service"]
