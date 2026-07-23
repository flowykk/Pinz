package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	redis "github.com/redis/go-redis/v9"
)

// InitRedisClient creates a Redis client for API Gateway. Returns (nil, nil) if
// REDIS_ADDR and REDIS_URL are both empty (optional Redis).
func InitRedisClient() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		if u := os.Getenv("REDIS_URL"); u != "" {
			opt, err := redis.ParseURL(u)
			if err != nil {
				return nil, err
			}
			client := redis.NewClient(opt)
			if err := client.Ping(context.Background()).Err(); err != nil {
				return nil, fmt.Errorf("redis ping: %w", err)
			}
			instrumentRedis(client)
			return client, nil
		}
		return nil, nil
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	instrumentRedis(client)
	return client, nil
}

func instrumentRedis(client *redis.Client) {
	if err := redisotel.InstrumentTracing(client); err != nil {
		slog.Warn("redis tracing instrumentation failed", "error", err)
	}
	if err := redisotel.InstrumentMetrics(client); err != nil {
		slog.Warn("redis metrics instrumentation failed", "error", err)
	}
}
