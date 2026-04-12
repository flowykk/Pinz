package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"pinz/backend/auth-service/internal/email"
	"pinz/backend/auth-service/internal/repositories"
	"pinz/backend/auth-service/internal/worker"
)

func main() {
	slog.Info("auth email-worker starting")
	_ = godotenv.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	redisClient, err := repositories.InitRedisClient()
	if err != nil {
		slog.Error("email-worker: failed to init redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	sender := email.NewSender(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USERNAME"),
		os.Getenv("SMTP_PASSWORD"),
		os.Getenv("SMTP_FROM"),
	)

	if err := worker.Run(ctx, redisClient, sender); err != nil {
		slog.Error("email-worker: run failed", "error", err)
		os.Exit(1)
	}
}
