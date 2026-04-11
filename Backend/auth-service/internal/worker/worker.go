package worker

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/auth-service/internal/email"
)

const (
	emailStream        = "pinz:auth:email:tasks"
	emailConsumerGroup = "auth-email-worker"
	emailConsumerName  = "auth-email-1"
)

func Run(ctx context.Context, redisClient *redis.Client, sender *email.Sender) error {
	if redisClient == nil || sender == nil {
		slog.Warn("email-worker: redis or sender not configured, worker disabled")
		<-ctx.Done()
		return nil
	}

	if err := ensureConsumerGroup(ctx, redisClient); err != nil {
		return err
	}

	slog.Info("email-worker: started", "stream", emailStream, "group", emailConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			slog.Info("email-worker: context cancelled, stopping")
			return nil
		default:
		}

		streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    emailConsumerGroup,
			Consumer: emailConsumerName,
			Streams:  []string{emailStream, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()
		if err != nil && err != redis.Nil {
			slog.WarnContext(ctx, "email-worker: XReadGroup error", "error", err)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				emailAddr, _ := msg.Values["email"].(string)
				code, _ := msg.Values["code"].(string)
				regID, _ := msg.Values["registration_id"].(string)

				if emailAddr == "" || code == "" {
					slog.WarnContext(ctx, "email-worker: missing email or code in message", "msg_id", msg.ID)
					_ = redisClient.XAck(ctx, emailStream, emailConsumerGroup, msg.ID)
					continue
				}

				if err := sender.SendVerificationCode(emailAddr, code); err != nil {
					slog.ErrorContext(ctx, "email-worker: failed to send email",
						"registration_id", regID, "email", emailAddr, "error", err)
				} else {
					slog.InfoContext(ctx, "email-worker: verification email sent",
						"registration_id", regID, "email", emailAddr)
				}

				_ = redisClient.XAck(ctx, emailStream, emailConsumerGroup, msg.ID)
			}
		}
	}
}

func ensureConsumerGroup(ctx context.Context, client *redis.Client) error {
	if err := client.XGroupCreateMkStream(ctx, emailStream, emailConsumerGroup, "0").Err(); err != nil && !isBusyGroupErr(err) {
		slog.ErrorContext(ctx, "email-worker: failed to create consumer group", "error", err)
		return err
	}
	return nil
}

func isBusyGroupErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}
