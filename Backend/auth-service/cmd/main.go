package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"

	"pinz/backend/auth-service/internal/db"
	"pinz/backend/auth-service/internal/di"
	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/server"
	pinzotel "pinz/backend/pkg/otel"
)

func main() {
	slog.Info("auth-service starting")
	_ = godotenv.Load()

	ctx := context.Background()

	otelProviders, err := pinzotel.Init(ctx, "auth-service", "1.0.0")
	if err != nil {
		slog.Warn("OTel init failed, running without telemetry", "error", err)
	} else {
		defer func() {
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			otelProviders.Shutdown(shutCtx)
		}()
		slog.SetDefault(slog.New(otelslog.NewHandler("auth-service")))
		if err := runtimemetrics.Start(
			runtimemetrics.WithMinimumReadMemStatsInterval(15 * time.Second),
		); err != nil {
			slog.Warn("runtime metrics start failed", "error", err)
		}
	}

	sqlDB, err := db.InitDB()
	if err != nil {
		slog.Error("db init failed", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	redisClient, err := repositories.InitRedisClient()
	if err != nil {
		slog.Error("redis init failed", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	deps, err := di.BuildDependencies(sqlDB, redisClient)
	if err != nil {
		slog.Error("failed to build dependencies", "error", err)
		os.Exit(1)
	}

	// email-воркер перенесён в notification-service. Здесь auth-service
	// только публикует задачи в pinz:auth:email:tasks; отправка выполняется
	// notification-service'ом.

	if err := server.RunGRPCServer(deps.AuthService); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
