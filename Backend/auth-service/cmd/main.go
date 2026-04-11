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
	"pinz/backend/auth-service/internal/email"
	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/server"
	"pinz/backend/auth-service/internal/worker"
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

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	sender := email.NewSender(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USERNAME"),
		os.Getenv("SMTP_PASSWORD"),
		os.Getenv("SMTP_FROM"),
	)
	go func() {
		if err := worker.Run(workerCtx, redisClient, sender); err != nil {
			slog.Error("email worker error", "error", err)
		}
	}()

	if err := server.RunGRPCServer(deps.AuthService); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
