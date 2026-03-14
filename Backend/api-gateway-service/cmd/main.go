// @title Pinz API Gateway
// @version 1.0
// @description API Gateway for Pinz mobile client
// @host pinz.website
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT access token.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"

	"pinz/backend/api-gateway-service/internal/di"
	"pinz/backend/api-gateway-service/internal/server"
	pinzotel "pinz/backend/pkg/otel"

	docs "pinz/backend/api-gateway-service/docs"
)

func main() {
	_ = godotenv.Load()

	if host := os.Getenv("SWAGGER_HOST"); host != "" {
		docs.SwaggerInfo.Host = host
	}

	ctx := context.Background()

	otelProviders, err := pinzotel.Init(ctx, "api-gateway-service", "1.0.0")
	if err != nil {
		slog.Warn("OTel init failed, running without telemetry", "error", err)
	} else {
		defer func() {
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			otelProviders.Shutdown(shutCtx)
		}()
		slog.SetDefault(slog.New(otelslog.NewHandler("api-gateway-service")))
		if err := runtimemetrics.Start(
			runtimemetrics.WithMinimumReadMemStatsInterval(15 * time.Second),
		); err != nil {
			slog.Warn("runtime metrics start failed", "error", err)
		}
	}

	deps, err := di.BuildDependencies()
	if err != nil {
		slog.Error("failed to build dependencies", "error", err)
		os.Exit(1)
	}
	defer deps.Close()

	srv := server.NewServer(deps)
	if err := srv.Run(); err != nil {
		slog.Error("server run error", "error", err)
		os.Exit(1)
	}
}
