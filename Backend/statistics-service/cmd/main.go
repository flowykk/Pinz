package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"

	pinzotel "pinz/backend/pkg/otel"
	"pinz/backend/statistics-service/internal/db"
	"pinz/backend/statistics-service/internal/di"
	"pinz/backend/statistics-service/internal/repositories"
	"pinz/backend/statistics-service/internal/server"
	"pinz/backend/statistics-service/internal/worker"
)

func main() {
	slog.Info("statistics-service starting")
	_ = godotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		slog.Warn("OTEL_SDK_DISABLED=true — skipping telemetry initialization")
	} else {
		otelProviders, err := pinzotel.Init(ctx, "statistics-service", "1.0.0")
		if err != nil {
			slog.Warn("OTel init failed, running without telemetry", "error", err)
		} else {
			defer func() {
				shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutCancel()
				otelProviders.Shutdown(shutCtx)
			}()
			slog.SetDefault(slog.New(otelslog.NewHandler("statistics-service")))
			if err := runtimemetrics.Start(
				runtimemetrics.WithMinimumReadMemStatsInterval(15 * time.Second),
			); err != nil {
				slog.Warn("runtime metrics start failed", "error", err)
			}
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
		slog.Warn("redis not available, events consumer disabled", "error", err)
	} else if rc != nil {
		redisClient = rc
		defer redisClient.Close()
	}

	slog.Info("building dependencies")
	deps, err := di.BuildDependencies(sqlDB, redisClient)
	if err != nil {
		slog.Error("failed to build dependencies", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := worker.Run(ctx, deps.WorkerDeps); err != nil {
			slog.Error("worker stopped with error", "error", err)
		}
	}()

	slog.Info("dependencies ready, starting gRPC server")
	if err := server.RunGRPCServer(deps.StatisticsService); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
