package apns

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"

	"pinz/backend/notification-service/internal/metrics"
	"pinz/backend/notification-service/internal/models"
)

// Sender — интерфейс для отправки APNS push, чтобы worker/scheduler можно было
// тестировать с fake-реализацией без настоящей Apple-ки.
type Sender interface {
	Send(ctx context.Context, apnsToken string, n models.PushNotification) error
}

// Client — боевая реализация через github.com/sideshow/apns2.
type Client struct {
	cli *apns2.Client
	bundleID string
}

// NewClientFromEnv строит клиент из env. Возвращает (nil, nil) если ключи
// APNS не заданы — сервис в этом случае работает без отправки push.
// Требуемые env: APNS_KEY_ID, APNS_TEAM_ID, APNS_BUNDLE_ID, APNS_KEY_BASE64.
// APNS_PRODUCTION=true переключает на production Apple-хост (по умолчанию sandbox).
func NewClientFromEnv() (*Client, error) {
	keyID := os.Getenv("APNS_KEY_ID")
	teamID := os.Getenv("APNS_TEAM_ID")
	bundleID := os.Getenv("APNS_BUNDLE_ID")
	keyB64 := os.Getenv("APNS_KEY_BASE64")
	if keyID == "" || teamID == "" || bundleID == "" || keyB64 == "" {
		slog.Warn("apns: credentials not set, push sender disabled")
		return nil, nil
	}
	keyBytes, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("apns: decode key base64: %w", err)
	}
	authKey, err := token.AuthKeyFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("apns: parse auth key: %w", err)
	}
	tok := &token.Token{AuthKey: authKey, KeyID: keyID, TeamID: teamID}
	cli := apns2.NewTokenClient(tok)
	if os.Getenv("APNS_PRODUCTION") == "true" {
		cli = cli.Production()
	} else {
		cli = cli.Development()
	}
	return &Client{cli: cli, bundleID: bundleID}, nil
}

func (c *Client) Send(ctx context.Context, apnsToken string, n models.PushNotification) error {
	if c == nil || c.cli == nil {
		return nil
	}
	pl := payload.NewPayload().AlertTitle(n.Title).AlertBody(n.Body).Sound("default")
	for k, v := range n.Extra {
		pl = pl.Custom(k, v)
	}
	notif := &apns2.Notification{
		DeviceToken: apnsToken,
		Topic: c.bundleID,
		Payload: pl,
	}
	eventType := ""
	if n.Extra != nil {
		eventType = n.Extra["event_type"]
	}
	start := time.Now()
	res, err := c.cli.PushWithContext(ctx, notif)
	dur := time.Since(start).Seconds()
	if err != nil {
		metrics.APNSPush(ctx, eventType, "transport_error")
		metrics.ObserveAPNSDuration(ctx, dur, "transport_error")
		return fmt.Errorf("apns push: %w", err)
	}
	if !res.Sent() {
		reason := res.Reason
		if reason == "" {
			reason = "status_" + strconv.Itoa(res.StatusCode)
		}
		metrics.APNSPush(ctx, eventType, reason)
		metrics.ObserveAPNSDuration(ctx, dur, "rejected")
		return fmt.Errorf("apns rejected: status=%d reason=%s", res.StatusCode, res.Reason)
	}
	metrics.APNSPush(ctx, eventType, "sent")
	metrics.ObserveAPNSDuration(ctx, dur, "sent")
	return nil
}

// ErrNotConfigured возвращается, когда отправку пытаются сделать без настроенного клиента.
var ErrNotConfigured = errors.New("apns: not configured")
