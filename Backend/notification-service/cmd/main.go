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
	"go.opentelemetry.io/otel"

	pinzotel "pinz/backend/pkg/otel"
	"pinz/backend/notification-service/internal/apns"
	"pinz/backend/notification-service/internal/db"
	"pinz/backend/notification-service/internal/di"
	"pinz/backend/notification-service/internal/email"
	"pinz/backend/notification-service/internal/metrics"
	"pinz/backend/notification-service/internal/repositories"
	"pinz/backend/notification-service/internal/scheduler"
	"pinz/backend/notification-service/internal/server"
	"pinz/backend/notification-service/internal/worker"
)

func main() {
	slog.Info("notification-service starting")
	_ = godotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		slog.Warn("OTEL_SDK_DISABLED=true — skipping telemetry initialization")
	} else {
		otelProviders, err := pinzotel.Init(ctx, "notification-service", "1.0.0")
		if err != nil {
			slog.Warn("OTel init failed, running without telemetry", "error", err)
		} else {
			defer func() {
				shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutCancel()
				otelProviders.Shutdown(shutCtx)
			}()
			slog.SetDefault(slog.New(otelslog.NewHandler("notification-service")))
			if err := runtimemetrics.Start(
				runtimemetrics.WithMinimumReadMemStatsInterval(15 * time.Second),
			); err != nil {
				slog.Warn("runtime metrics start failed", "error", err)
			}
			metrics.Init()
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
	if err := pinzotel.RegisterDBPoolMetrics(otel.Meter("notification-service"), sqlDB, "pinz_notifications"); err != nil {
		slog.Warn("db pool metrics registration failed", "error", err)
	}

	var redisClient *redis.Client
	if rc, err := repositories.InitRedisClient(); err != nil {
		slog.Warn("redis not available, streams consumers and scheduler disabled", "error", err)
	} else if rc != nil {
		redisClient = rc
		defer redisClient.Close()
		if err := pinzotel.RegisterStreamMetrics(otel.Meter("notification-service"), repositories.StreamQueryer{Client: redisClient}, []pinzotel.StreamSpec{
			{Stream: worker.EmailStream, Groups: []string{"notification-email-worker"}},
			{Stream: worker.TripEventsStream, Groups: []string{"notification-service-trip"}},
		}); err != nil {
			slog.Warn("stream metrics registration failed", "error", err)
		}
	}

	apnsClient, err := apns.NewClientFromEnv()
	if err != nil {
		slog.Error("apns init failed", "error", err)
		os.Exit(1)
	}
	var apnsSender apns.Sender
	if apnsClient != nil {
		apnsSender = apnsClient
	}

	emailSender := email.NewSenderFromEnv()
	if emailSender == nil {
		slog.Warn("smtp not configured (SMTP_HOST empty), email-worker will ack without sending")
	}

	slog.Info("building dependencies")
	deps, err := di.BuildDependencies(sqlDB, redisClient, apnsSender, emailSender)
	if err != nil {
		slog.Error("failed to build dependencies", "error", err)
		os.Exit(1)
	}
	defer func() {
		if deps.TripClient != nil {
			_ = deps.TripClient.Close()
		}
	}()

	go func() {
		if err := worker.RunTripEvents(ctx, deps.TripEventsDeps); err != nil {
			slog.Error("trip-events worker stopped with error", "error", err)
		}
	}()
	go func() {
		if err := worker.RunEmail(ctx, deps.EmailDeps); err != nil {
			slog.Error("email worker stopped with error", "error", err)
		}
	}()
	go func() {
		if err := scheduler.Run(ctx, deps.SchedulerDeps); err != nil {
			slog.Error("scheduler stopped with error", "error", err)
		}
	}()

	slog.Info("dependencies ready, starting gRPC server")
	if err := server.RunGRPCServer(deps.NotificationService); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
