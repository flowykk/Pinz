package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"

	pinzotel "pinz/backend/pkg/otel"
	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/di"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/server"
)

func main() {
	slog.Info("trip-service starting")
	_ = godotenv.Load()

	ctx := context.Background()

	otelProviders, err := pinzotel.Init(ctx, "trip-service", "1.0.0")
	if err != nil {
		slog.Warn("OTel init failed, running without telemetry", "error", err)
	} else {
		defer func() {
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			otelProviders.Shutdown(shutCtx)
		}()
		slog.SetDefault(slog.New(otelslog.NewHandler("trip-service")))
		if err := runtimemetrics.Start(
			runtimemetrics.WithMinimumReadMemStatsInterval(15 * time.Second),
		); err != nil {
			slog.Warn("runtime metrics start failed", "error", err)
		}
	}

	slog.Info("connecting to database")
	sqlDB, err := db.InitDB()
	if err != nil {
		slog.Error("db init failed", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()
	slog.Info("database ready")

	var redisClient *redis.Client
	if rc, err := repositories.InitRedisClient(); err != nil {
		slog.Warn("redis not available, trip events will not be published", "error", err)
	} else if rc != nil {
		redisClient = rc
		defer redisClient.Close()
	}

	slog.Info("building dependencies")
	deps, err := di.BuildDependencies(ctx, sqlDB, redisClient)
	if err != nil {
		slog.Error("failed to build dependencies", "error", err)
		os.Exit(1)
	}
	slog.Info("dependencies ready, starting gRPC server")
	if err := server.RunGRPCServer(deps.TripService); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
