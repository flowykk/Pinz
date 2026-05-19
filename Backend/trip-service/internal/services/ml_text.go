package services

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"pinz/backend/trip-service/internal/metrics"
	"pinz/backend/trip-service/internal/repositories"
)

const mlTextTaskTTL = 30 * time.Minute

// nil pointer = поле не менялось; пустая/whitespace строка отсекается TrimSpace и в ML не уходит.
func PublishTripTextModeration(ctx context.Context, broker repositories.MLBroker, tripID string, name, description *string) {
	if broker == nil || tripID == "" {
		return
	}
	items := make([]repositories.MLTextItem, 0, 2)
	if name != nil && strings.TrimSpace(*name) != "" {
		items = append(items, repositories.MLTextItem{
			ItemID:     uuid.NewString(),
			EntityKind: repositories.MLTextEntityTrip,
			EntityID:   tripID,
			Field:      repositories.MLTextFieldName,
			Text:       *name,
		})
	}
	if description != nil && strings.TrimSpace(*description) != "" {
		items = append(items, repositories.MLTextItem{
			ItemID:     uuid.NewString(),
			EntityKind: repositories.MLTextEntityTrip,
			EntityID:   tripID,
			Field:      repositories.MLTextFieldDescription,
			Text:       *description,
		})
	}
	publishTextItems(ctx, broker, tripID, items)
}

func PublishPinTextModeration(ctx context.Context, broker repositories.MLBroker, tripID, pinID string, name, description *string) {
	if broker == nil || pinID == "" {
		return
	}
	items := make([]repositories.MLTextItem, 0, 2)
	if name != nil && strings.TrimSpace(*name) != "" {
		items = append(items, repositories.MLTextItem{
			ItemID:     uuid.NewString(),
			EntityKind: repositories.MLTextEntityPin,
			EntityID:   pinID,
			Field:      repositories.MLTextFieldName,
			Text:       *name,
		})
	}
	if description != nil && strings.TrimSpace(*description) != "" {
		items = append(items, repositories.MLTextItem{
			ItemID:     uuid.NewString(),
			EntityKind: repositories.MLTextEntityPin,
			EntityID:   pinID,
			Field:      repositories.MLTextFieldDescription,
			Text:       *description,
		})
	}
	publishTextItems(ctx, broker, tripID, items)
}

type PinNameItem struct {
	PinID string
	Name  string
}

func PublishPinNameBatch(ctx context.Context, broker repositories.MLBroker, tripID string, pins []PinNameItem) {
	if broker == nil || len(pins) == 0 {
		return
	}
	items := make([]repositories.MLTextItem, 0, len(pins))
	for _, pn := range pins {
		if pn.PinID == "" || strings.TrimSpace(pn.Name) == "" {
			continue
		}
		items = append(items, repositories.MLTextItem{
			ItemID:     uuid.NewString(),
			EntityKind: repositories.MLTextEntityPin,
			EntityID:   pn.PinID,
			Field:      repositories.MLTextFieldName,
			Text:       pn.Name,
		})
	}
	publishTextItems(ctx, broker, tripID, items)
}

func publishTextItems(ctx context.Context, broker repositories.MLBroker, tripID string, items []repositories.MLTextItem) {
	if len(items) == 0 {
		return
	}
	msg := repositories.MLTextTaskMessage{
		Flow:          repositories.MLFlowTextModeration,
		TripID:        tripID,
		Items:         items,
		ExpiresAtUnix: time.Now().Add(mlTextTaskTTL).Unix(),
	}
	if err := broker.PublishMLTextTask(ctx, msg); err != nil {
		slog.WarnContext(ctx, "ml text task: publish failed", "trip_id", tripID, "items", len(items), "error", err)
		return
	}
	for _, it := range items {
		metrics.MLTextTaskItem(ctx, it.EntityKind, it.Field)
	}
}
