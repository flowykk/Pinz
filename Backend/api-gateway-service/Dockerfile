FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o api-gateway-service ./cmd/main.go

FROM scratch
COPY --from=builder /app/api-gateway-service /app/api-gateway-service
EXPOSE 8080
ENTRYPOINT ["/app/api-gateway-service"]
