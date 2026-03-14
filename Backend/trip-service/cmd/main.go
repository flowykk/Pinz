package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"

	pinzotel "pinz/backend/pkg/otel"
	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/di"
	"pinz/backend/trip-service/internal/server"
)

func main() {
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

	sqlDB, err := db.InitDB()
	if err != nil {
		slog.Error("db init failed", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	deps, err := di.BuildDependencies(sqlDB)
	if err != nil {
		slog.Error("failed to build dependencies", "error", err)
		os.Exit(1)
	}
	if err := server.RunGRPCServer(deps.TripService); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
