package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"

	pinzotel "pinz/backend/pkg/otel"
	"pinz/backend/trip-service/internal/db"
	"pinz/backend/trip-service/internal/di"
	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/repositories"
	"pinz/backend/trip-service/internal/server"
	"pinz/backend/trip-service/internal/worker"
)

func main() {
	slog.Info("trip-service starting")
	_ = godotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	otelProviders, err := pinzotel.Init(ctx, "trip-service", "1.0.0")
	if err != nil {
		slog.Warn("OTel init failed, running without telemetry", "error", err)
	} else {
		defer func() {
			shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutCancel()
			otelProviders.Shutdown(shutCtx)
		}()
		slog.SetDefault(slog.New(otelslog.NewHandler("trip-service")))
		if err := runtimemetrics.Start(
			runtimemetrics.WithMinimumReadMemStatsInterval(15 * time.Second),
		); err != nil {
			slog.Warn("runtime metrics start failed", "error", err)
		}
		metrics.Init()
	}

	slog.Info("connecting to database")
	sqlDB, err := db.InitDB()
	if err != nil {
		slog.Error("db init failed", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()
	slog.Info("database ready")
	if err := pinzotel.RegisterDBPoolMetrics(otel.Meter("trip-service"), sqlDB, "pinz_trips"); err != nil {
		slog.Warn("db pool metrics registration failed", "error", err)
	}

	var redisClient *redis.Client
	if rc, err := repositories.InitRedisClient(); err != nil {
		slog.Warn("redis not available, trip events will not be published", "error", err)
	} else if rc != nil {
		redisClient = rc
		defer redisClient.Close()
		if err := pinzotel.RegisterStreamMetrics(otel.Meter("trip-service"), repositories.StreamQueryer{Client: redisClient}, []pinzotel.StreamSpec{
			{Stream: "pinz:trip:ml:tasks", Groups: []string{"trip-service-worker"}},
			{Stream: "pinz:trip:ml:results", Groups: []string{"trip-service-ml-results"}},
			{Stream: "pinz:trip:pin_upload:tasks", Groups: []string{"trip-service-pin-upload"}},
			{Stream: "pinz:trip:privacy:events", Groups: []string{"trip-service-privacy"}},
			{Stream: "pinz:trip:geo_events", Groups: []string{"trip-service-geo-worker"}},
			{Stream: "pinz:trip:events", Groups: nil},
			{Stream: "pinz:stats:events", Groups: nil},
		}); err != nil {
			slog.Warn("stream metrics registration failed", "error", err)
		}
	}

	slog.Info("building dependencies")
	deps, err := di.BuildDependencies(ctx, sqlDB, redisClient)
	if err != nil {
		slog.Error("failed to build dependencies", "error", err)
		os.Exit(1)
	}

	// Start worker as a background goroutine.
	go func() {
		if err := worker.Run(ctx, deps.RedisClient, deps.TripRepo, deps.ParticipantRepo, deps.MediaRepo, deps.TagRepo, deps.PinRepo, deps.EventRepo, deps.TripPrivacyRepo, deps.PinPrivacyRepo, deps.MediaPrivacyRepo); err != nil {
			slog.Error("worker stopped with error", "error", err)
		}
	}()

	// Geo consumer: pinz:trip:geo_events ← statistics-service (PIN_LOCATIONS_RESOLVED).
	go func() {
		if err := worker.RunGeoConsumer(ctx, deps.RedisClient, deps.GeoRepo, deps.PinRepo, deps.GeoEventLogRepo); err != nil {
			slog.Error("geo consumer stopped with error", "error", err)
		}
	}()

	// cron для закрытия заброшенных add-media сессий (72ч без активности).
	go worker.RunSessionCleanup(ctx, deps.AddMediaSessionRepo, deps.TripRepo, deps.ParticipantRepo, deps.MediaRepo, deps.EventRepo, deps.MediaURLs)

	// cron для закрытия заброшенных pin_upload сессий.
	go worker.RunPinUploadCleanup(ctx, deps.PinUploadSessionRepo, deps.MediaRepo, deps.MediaURLs)

	// cron для очистки generated-трипов рекомендаций, выкинутых из favourites.
	go worker.RunGeneratedTripCleanup(ctx, deps.TripRepo)

	// Унифицированный pin-upload consumer (pinz:trip:pin_upload:tasks). N горутин в одной group.
	for i := 0; i < worker.PinUploadConsumerCount; i++ {
		consumerName := "trip-pin-upload-" + strconv.Itoa(i)
		go worker.RunPinUploadConsumer(ctx, deps.RedisClient, consumerName, worker.PinUploadConsumerDeps{
			SessionRepo: deps.PinUploadSessionRepo,
			MediaRepo:   deps.MediaRepo,
			EventRepo:   deps.EventRepo,
			MediaURLs:   deps.MediaURLs,
		})
	}

	slog.Info("dependencies ready, starting gRPC server")
	if err := server.RunGRPCServer(deps.TripService); err != nil {
		slog.Error("gRPC server error", "error", err)
		os.Exit(1)
	}
}
