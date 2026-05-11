package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/notification-service/internal/apns"
	"pinz/backend/notification-service/internal/metrics"
	"pinz/backend/notification-service/internal/models"
	"pinz/backend/notification-service/internal/repositories"
)

const (
	TripEventsStream = "pinz:trip:events"
	tripEventsConsumerGroup = "notification-service-trip"
	tripEventsConsumerName = "notif-trip-1"
)

// Event types, публикуемые trip-service (repositories/redis_repository.go:PublishTripEvent).
const (
	EventParticipantJoined = "PARTICIPANT_JOINED"
	EventParticipantLeft = "PARTICIPANT_LEFT"
	EventParticipantRemoved = "PARTICIPANT_REMOVED"
	EventAdminChanged = "ADMIN_CHANGED"
	EventTripReady = "TRIP_READY"
	EventPinAdded = "PIN_ADDED"
	EventAddMediaSessionCompleted = "ADD_MEDIA_SESSION_COMPLETED" //
)

type TripEventsDeps struct {
	Redis *redis.Client
	Tokens repositories.DeviceTokensRepositoryInterface
	NotifLog repositories.NotificationLogRepositoryInterface
	TripClient repositories.TripClientInterface
	APNS apns.Sender
}

// RunTripEvents — consumer-loop для pinz:trip:events. Преобразует события в
// APNS push-уведомления, учитывая notifications_enabled.
func RunTripEvents(ctx context.Context, d TripEventsDeps) error {
	if d.Redis == nil {
		slog.Warn("trip-events worker: redis not configured, consumer disabled")
		<-ctx.Done()
		return nil
	}
	if err := ensureGroup(ctx, d.Redis, TripEventsStream, tripEventsConsumerGroup); err != nil {
		return err
	}
	slog.Info("trip-events worker: started", "stream", TripEventsStream, "group", tripEventsConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			slog.Info("trip-events worker: context cancelled, stopping")
			return nil
		default:
		}
		streams, err := d.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: tripEventsConsumerGroup,
			Consumer: tripEventsConsumerName,
			Streams: []string{TripEventsStream, ">"},
			Count: 50,
			Block: 2 * time.Second,
		}).Result()
		if err != nil && err != redis.Nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.WarnContext(ctx, "trip-events worker: XReadGroup error", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				start := time.Now()
				eventType, _ := msg.Values["event_type"].(string)
				if err := handleTripEvent(ctx, d, msg); err != nil {
					slog.WarnContext(ctx, "trip-events worker: handle failed, skipping ack",
						"msg_id", msg.ID, "error", err)
					metrics.StreamConsumed(ctx, TripEventsStream, tripEventsConsumerGroup, eventType, "error")
					continue
				}
				metrics.StreamConsumed(ctx, TripEventsStream, tripEventsConsumerGroup, eventType, "success")
				metrics.ObserveStreamConsumeDuration(ctx, time.Since(start).Seconds(), TripEventsStream, tripEventsConsumerGroup)
				if err := d.Redis.XAck(ctx, stream.Stream, tripEventsConsumerGroup, msg.ID).Err(); err != nil {
					slog.WarnContext(ctx, "trip-events worker: XAck failed", "msg_id", msg.ID, "error", err)
				}
			}
		}
	}
}

func handleTripEvent(ctx context.Context, d TripEventsDeps, msg redis.XMessage) error {
	eventType, _ := msg.Values["event_type"].(string)
	tripID, _ := msg.Values["trip_id"].(string)
	actorUserID, _ := msg.Values["user_id"].(string)
	if eventType == "" || tripID == "" {
		return nil
	}

	recipients, err := resolveRecipients(ctx, d, eventType, tripID, actorUserID)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	if len(recipients) == 0 {
		return nil
	}

	settings, err := d.TripClient.GetNotificationSettings(ctx, tripID, recipients)
	if err != nil {
		slog.WarnContext(ctx, "trip-events worker: GetNotificationSettings failed, defaulting to all enabled", "error", err)
		settings = map[string]bool{}
		for _, u := range recipients {
			settings[u] = true
		}
	}
	enabled := make([]string, 0, len(recipients))
	for _, u := range recipients {
		if settings[u] {
			enabled = append(enabled, u)
		}
	}
	if len(enabled) == 0 {
		return nil
	}

	tokens, err := d.Tokens.ListByUsers(ctx, enabled)
	if err != nil {
		return fmt.Errorf("list device_tokens: %w", err)
	}
	if len(tokens) == 0 {
		return nil
	}

	title, body := buildMessage(eventType, tripID)
	push := models.PushNotification{
		Title: title,
		Body: body,
		Extra: map[string]string{"trip_id": tripID, "event_type": eventType},
	}

	for _, t := range tokens {
		if alreadySent, _ := d.NotifLog.IsSent(ctx, msg.ID, t.APNSToken); alreadySent {
			continue
		}
		if d.APNS == nil {
			continue
		}
		if err := d.APNS.Send(ctx, t.APNSToken, push); err != nil {
			slog.WarnContext(ctx, "trip-events worker: APNS send failed",
				"apns_token", maskToken(t.APNSToken), "error", err)
			continue
		}
		if _, err := d.NotifLog.MarkSent(ctx, msg.ID, t.APNSToken); err != nil {
			slog.WarnContext(ctx, "trip-events worker: MarkSent failed", "error", err)
		}
	}
	return nil
}

// resolveRecipients — кому адресовано событие:
// PARTICIPANT_JOINED: все текущие участники кроме присоединившегося (actor).
// PARTICIPANT_LEFT: все остальные участники (actor уже удалён из списка).
// PARTICIPANT_REMOVED: все остальные участники + сам удалённый (actor).
// После удаления actor отсутствует в списке участников — добавляем его руками.
// ADMIN_CHANGED: все участники трипа (включая нового админа = actor).
// TRIP_READY: owner (actor содержит owner_user_id — см. trip_service.go).
// PIN_ADDED: все участники кроме автора (actor).
// ADD_MEDIA_SESSION_COMPLETED: все участники кроме инициатора Confirm (actor) — M1.
func resolveRecipients(ctx context.Context, d TripEventsDeps, eventType, tripID, actorUserID string) ([]string, error) {
	participants, err := d.TripClient.ListTripParticipantIDs(ctx, tripID)
	if err != nil {
		return nil, err
	}
	switch eventType {
	case EventAdminChanged:
		return participants, nil
	case EventTripReady:
		if actorUserID != "" {
			return []string{actorUserID}, nil
		}
		return participants, nil
	case EventParticipantRemoved:
		out := make([]string, 0, len(participants)+1)
		for _, u := range participants {
			if u != actorUserID {
				out = append(out, u)
			}
		}
		if actorUserID != "" {
			out = append(out, actorUserID)
		}
		return out, nil
	default:
		// PARTICIPANT_JOINED / PARTICIPANT_LEFT / PIN_ADDED: без actor.
		out := make([]string, 0, len(participants))
		for _, u := range participants {
			if u != actorUserID {
				out = append(out, u)
			}
		}
		return out, nil
	}
}

func buildMessage(eventType, tripID string) (title, body string) {
	switch eventType {
	case EventParticipantJoined:
		return "Pinz", "New participant joined your trip"
	case EventParticipantLeft:
		return "Pinz", "A participant left the trip"
	case EventParticipantRemoved:
		return "Pinz", "A participant was removed from the trip"
	case EventAdminChanged:
		return "Pinz", "You are now the admin of a trip"
	case EventTripReady:
		return "Pinz", "Your trip is ready"
	case EventPinAdded:
		return "Pinz", "New pin added to your trip"
	case EventAddMediaSessionCompleted:
		// пуш другим participant'ам о завершении add-media сессии.
		// Автор Confirm'а исключается в resolveRecipients (M1).
		return "Pinz", "New pins were added to the trip"
	default:
		return "Pinz", "Trip update"
	}
}

func ensureGroup(ctx context.Context, client *redis.Client, stream, group string) error {
	if err := client.XGroupCreateMkStream(ctx, stream, group, "0").Err(); err != nil && !isBusyGroupErr(err) {
		slog.ErrorContext(ctx, "worker: create consumer group failed", "stream", stream, "error", err)
		return err
	}
	return nil
}

func isBusyGroupErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "BUSYGROUP")
}

func maskToken(t string) string {
	if len(t) <= 8 {
		return "***"
	}
	return t[:4] + "..." + t[len(t)-4:]
}
