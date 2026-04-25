package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/notification-service/internal/email"
)

const (
	EmailStream = "pinz:auth:email:tasks"
	emailConsumerGroup = "notification-email-worker"
	emailConsumerName = "notif-email-1"
)

type EmailDeps struct {
	Redis *redis.Client
	Sender *email.Sender
}

// RunEmail — перенос auth-service/internal/worker/worker.go. Формат сообщений
// не менялся: auth-service публикует email+code+registration_id, мы отправляем
// через SMTP. Если SMTP не настроен (SMTP_HOST пуст) — XAck без отправки,
// чтобы стрим не распухал.
func RunEmail(ctx context.Context, d EmailDeps) error {
	if d.Redis == nil {
		slog.Warn("email worker: redis not configured, consumer disabled")
		<-ctx.Done()
		return nil
	}
	if err := ensureGroup(ctx, d.Redis, EmailStream, emailConsumerGroup); err != nil {
		return err
	}
	slog.Info("email worker: started", "stream", EmailStream, "group", emailConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			slog.Info("email worker: context cancelled, stopping")
			return nil
		default:
		}
		streams, err := d.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: emailConsumerGroup,
			Consumer: emailConsumerName,
			Streams: []string{EmailStream, ">"},
			Count: 10,
			Block: 2 * time.Second,
		}).Result()
		if err != nil && err != redis.Nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.WarnContext(ctx, "email worker: XReadGroup error", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				emailAddr, _ := msg.Values["email"].(string)
				code, _ := msg.Values["code"].(string)
				regID, _ := msg.Values["registration_id"].(string)

				if emailAddr == "" || code == "" {
					slog.WarnContext(ctx, "email worker: missing email or code", "msg_id", msg.ID)
					_ = d.Redis.XAck(ctx, EmailStream, emailConsumerGroup, msg.ID)
					continue
				}

				if d.Sender == nil {
					slog.InfoContext(ctx, "email worker: SMTP not configured, skipping", "msg_id", msg.ID)
					_ = d.Redis.XAck(ctx, EmailStream, emailConsumerGroup, msg.ID)
					continue
				}

				if err := d.Sender.SendVerificationCode(emailAddr, code); err != nil {
					slog.ErrorContext(ctx, "email worker: send failed",
						"registration_id", regID, "email", emailAddr, "error", err)
				} else {
					slog.InfoContext(ctx, "email worker: verification email sent",
						"registration_id", regID, "email", emailAddr)
				}

				_ = d.Redis.XAck(ctx, EmailStream, emailConsumerGroup, msg.ID)
			}
		}
	}
}
