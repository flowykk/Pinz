package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"pinz/backend/notification-service/internal/apns"
	"pinz/backend/notification-service/internal/models"
	"pinz/backend/notification-service/internal/repositories"
	pb "pinz/backend/notification-service/pkg/proto"
)

const (
	lastRunKey = "pinz:notifications:scheduler:last_run"
	// Окно отправки. Scheduler просыпается раз в час, но отправляет только если
	// прошло ≥24ч с прошлого прогона (идемпотентность на рестартах пода).
	runInterval = 24 * time.Hour
	tickEvery   = 1 * time.Hour
)

type Deps struct {
	Redis      *redis.Client
	Tokens     repositories.DeviceTokensRepositoryInterface
	NotifLog   repositories.NotificationLogRepositoryInterface
	TripClient repositories.TripClientInterface
	APNS       apns.Sender
}

// Run запускает scheduler. Блокирующий вызов, возвращает nil при отмене ctx.
func Run(ctx context.Context, d Deps) error {
	if d.Redis == nil || d.TripClient == nil {
		slog.Warn("scheduler: redis or trip-client not configured, disabled")
		<-ctx.Done()
		return nil
	}
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()

	// Первый прогон сразу после старта (если не делали <24ч назад).
	runIfDue(ctx, d)
	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler: stopping")
			return nil
		case <-ticker.C:
			runIfDue(ctx, d)
		}
	}
}

func runIfDue(ctx context.Context, d Deps) {
	lastRunStr, err := d.Redis.Get(ctx, lastRunKey).Result()
	if err != nil && err != redis.Nil {
		slog.WarnContext(ctx, "scheduler: get last_run failed", "error", err)
		return
	}
	now := time.Now().UTC()
	if lastRunStr != "" {
		if prev, pErr := time.Parse(time.RFC3339, lastRunStr); pErr == nil {
			if now.Sub(prev) < runInterval {
				return
			}
		}
	}
	if err := d.Redis.Set(ctx, lastRunKey, now.Format(time.RFC3339), 7*24*time.Hour).Err(); err != nil {
		slog.WarnContext(ctx, "scheduler: set last_run failed", "error", err)
	}
	runOnce(ctx, d, now)
}

func runOnce(ctx context.Context, d Deps, now time.Time) {
	today := now.Unix()
	slog.InfoContext(ctx, "scheduler: starting run", "today_unix", today)

	trips, err := d.TripClient.ListAnniversaryTrips(ctx, today)
	if err != nil {
		slog.WarnContext(ctx, "scheduler: ListAnniversaryTrips failed", "error", err)
	} else {
		for _, t := range trips {
			body := buildAnniversaryBody(t)
			dispatchTrips(ctx, d, []*pb.NotificationTrip{t}, "TRIP_ANNIVERSARY", "Pinz", body)
		}
	}

	endedTrips, err := d.TripClient.ListEndedMonthAgoTrips(ctx, today)
	if err != nil {
		slog.WarnContext(ctx, "scheduler: ListEndedMonthAgoTrips failed", "error", err)
	} else {
		for _, t := range endedTrips {
			body := fmt.Sprintf("A month has passed since your trip %q ended", t.GetName())
			dispatchTrips(ctx, d, []*pb.NotificationTrip{t}, "TRIP_ENDED_MONTH_AGO", "Pinz", body)
		}
	}

	slog.InfoContext(ctx, "scheduler: run finished")
}

// buildAnniversaryBody — текст по ТЗ 11.3.1: «Вспомните, как это было:
// (год/ 2 года/ 5 лет) назад вы посетили [Название]».
func buildAnniversaryBody(t *pb.NotificationTrip) string {
	years := t.GetYearsElapsed()
	if years <= 0 {
		years = 1
	}
	return fmt.Sprintf("Remember: %d year(s) ago you visited %q", years, t.GetName())
}

func dispatchTrips(ctx context.Context, d Deps, trips []*pb.NotificationTrip, eventType, title, body string) {
	for _, t := range trips {
		recipients := t.GetParticipantUserIds()
		if len(recipients) == 0 {
			continue
		}
		settings, err := d.TripClient.GetNotificationSettings(ctx, t.GetTripId(), recipients)
		if err != nil {
			slog.WarnContext(ctx, "scheduler: GetNotificationSettings failed, defaulting enabled",
				"trip_id", t.GetTripId(), "error", err)
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
			continue
		}
		tokens, err := d.Tokens.ListByUsers(ctx, enabled)
		if err != nil {
			slog.WarnContext(ctx, "scheduler: ListByUsers failed",
				"trip_id", t.GetTripId(), "error", err)
			continue
		}
		if len(tokens) == 0 {
			continue
		}
		eventID := fmt.Sprintf("%s:%s:%d:%s", eventType, t.GetTripId(), t.GetYearsElapsed(), time.Now().UTC().Format("2006-01-02"))
		push := models.PushNotification{
			Title: title,
			Body:  body,
			Extra: map[string]string{"trip_id": t.GetTripId(), "event_type": eventType},
		}
		for _, tok := range tokens {
			if alreadySent, _ := d.NotifLog.IsSent(ctx, eventID, tok.APNSToken); alreadySent {
				continue
			}
			if d.APNS == nil {
				continue
			}
			if err := d.APNS.Send(ctx, tok.APNSToken, push); err != nil {
				slog.WarnContext(ctx, "scheduler: APNS send failed",
					"trip_id", t.GetTripId(), "error", err)
				continue
			}
			if _, err := d.NotifLog.MarkSent(ctx, eventID, tok.APNSToken); err != nil {
				slog.WarnContext(ctx, "scheduler: MarkSent failed", "error", err)
			}
		}
	}
}
