// Package metrics exposes lazy-init OTel instruments. Pre-Init calls are noop
// because the package-level vars stay nil until Init runs at startup.
package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tripsCreated metric.Int64Counter
	tripStatusTransitions metric.Int64Counter
	pinUploadSessions metric.Int64Counter
	pinUploadDuration metric.Float64Histogram
	streamConsumed metric.Int64Counter
	streamConsumeDuration metric.Float64Histogram
	streamPublished metric.Int64Counter
	recommendationSave metric.Int64Counter
	likes metric.Int64Counter
	battles metric.Int64Counter
	addMediaTakeovers metric.Int64Counter
	panics metric.Int64Counter
	mlPayloadSize metric.Int64Histogram
	mlPayloadMediaCount metric.Int64Histogram
	mlPresignDuration metric.Float64Histogram
	mlTextTaskItems metric.Int64Counter
	mlTextResultApplied metric.Int64Counter
	mlTextResultSkipped metric.Int64Counter
)

func Init() {
	m := otel.Meter("trip-service")

	tripsCreated, _ = m.Int64Counter("trips.created.total",
		metric.WithDescription("Trips created, labelled by flow (creation/recommendation_save)"),
	)
	tripStatusTransitions, _ = m.Int64Counter("trips.status_changed.total",
		metric.WithDescription("Trip status changes by destination status"),
	)
	pinUploadSessions, _ = m.Int64Counter("pin_upload.sessions.total",
		metric.WithDescription("Pin-upload session lifecycle events by stage and result"),
	)
	pinUploadDuration, _ = m.Float64Histogram("pin_upload.processing.duration_seconds",
		metric.WithDescription("Pin-upload async processing duration (consumer received → READY_FOR_REVIEW)"),
		metric.WithUnit("s"),
	)
	streamConsumed, _ = m.Int64Counter("stream.messages.consumed.total",
		metric.WithDescription("Redis stream messages consumed, labelled by stream/group/status"),
	)
	streamConsumeDuration, _ = m.Float64Histogram("stream.message.processing.duration_seconds",
		metric.WithDescription("Per-message processing duration"),
		metric.WithUnit("s"),
	)
	streamPublished, _ = m.Int64Counter("stream.messages.published.total",
		metric.WithDescription("Redis stream messages published (XADD), labelled by stream/event_type/result"),
	)
	recommendationSave, _ = m.Int64Counter("recommendations.save.total",
		metric.WithDescription("Recommendation save outcomes"),
	)
	likes, _ = m.Int64Counter("trip_reactions.total",
		metric.WithDescription("Trip reactions: like/dislike add/remove, favourite add/remove"),
	)
	battles, _ = m.Int64Counter("battles.total",
		metric.WithDescription("Battle lifecycle: started/result_submitted/finished"),
	)
	addMediaTakeovers, _ = m.Int64Counter("add_media.takeovers.total",
		metric.WithDescription("Cooperative add-media takeovers"),
	)
	panics, _ = m.Int64Counter("panics.total",
		metric.WithDescription("Recovered panics by component"),
	)
	mlPayloadSize, _ = m.Int64Histogram("ml.payload.size_bytes",
		metric.WithDescription("Size of ML task payload (pins/new_media JSON) in bytes"),
		metric.WithUnit("By"),
	)
	mlPayloadMediaCount, _ = m.Int64Histogram("ml.payload.media_count",
		metric.WithDescription("Number of media items in an ML task payload"),
	)
	mlPresignDuration, _ = m.Float64Histogram("ml.presign.generation_duration_seconds",
		metric.WithDescription("Duration of presigned GET URL generation when building ML payload"),
		metric.WithUnit("s"),
	)
	mlTextTaskItems, _ = m.Int64Counter("ml.text.task_items.total",
		metric.WithDescription("Text-moderation items published to ML, by entity_kind/field"),
	)
	mlTextResultApplied, _ = m.Int64Counter("ml.text.result_applied.total",
		metric.WithDescription("Text-moderation results applied to DB, by entity_kind/field/censored"),
	)
	mlTextResultSkipped, _ = m.Int64Counter("ml.text.result_skipped.total",
		metric.WithDescription("Text-moderation results skipped due to errors/unknown fields, by reason"),
	)
}

func TripCreated(ctx context.Context, flow string) {
	if tripsCreated == nil {
		return
	}
	tripsCreated.Add(ctx, 1, metric.WithAttributes(attribute.String("flow", flow)))
}

func TripStatusChanged(ctx context.Context, to string) {
	if tripStatusTransitions == nil {
		return
	}
	tripStatusTransitions.Add(ctx, 1, metric.WithAttributes(attribute.String("to", to)))
}

func PinUploadSession(ctx context.Context, stage, result string) {
	if pinUploadSessions == nil {
		return
	}
	pinUploadSessions.Add(ctx, 1, metric.WithAttributes(
		attribute.String("stage", stage),
		attribute.String("result", result),
	))
}

func ObservePinUploadDuration(ctx context.Context, seconds float64, result string) {
	if pinUploadDuration == nil {
		return
	}
	pinUploadDuration.Record(ctx, seconds, metric.WithAttributes(attribute.String("result", result)))
}

func StreamConsumed(ctx context.Context, stream, group, eventType, status string) {
	if streamConsumed == nil {
		return
	}
	streamConsumed.Add(ctx, 1, metric.WithAttributes(
		attribute.String("stream", stream),
		attribute.String("group", group),
		attribute.String("event_type", eventType),
		attribute.String("status", status),
	))
}

func ObserveStreamConsumeDuration(ctx context.Context, seconds float64, stream, group string) {
	if streamConsumeDuration == nil {
		return
	}
	streamConsumeDuration.Record(ctx, seconds, metric.WithAttributes(
		attribute.String("stream", stream),
		attribute.String("group", group),
	))
}

func StreamPublished(ctx context.Context, stream, eventType, result string) {
	if streamPublished == nil {
		return
	}
	streamPublished.Add(ctx, 1, metric.WithAttributes(
		attribute.String("stream", stream),
		attribute.String("event_type", eventType),
		attribute.String("result", result),
	))
}

func RecommendationSave(ctx context.Context, result string) {
	if recommendationSave == nil {
		return
	}
	recommendationSave.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func Reaction(ctx context.Context, kind, action string) {
	if likes == nil {
		return
	}
	likes.Add(ctx, 1, metric.WithAttributes(
		attribute.String("kind", kind),
		attribute.String("action", action),
	))
}

func Battle(ctx context.Context, stage string) {
	if battles == nil {
		return
	}
	battles.Add(ctx, 1, metric.WithAttributes(attribute.String("stage", stage)))
}

func AddMediaTakeover(ctx context.Context) {
	if addMediaTakeovers == nil {
		return
	}
	addMediaTakeovers.Add(ctx, 1)
}

func Panic(ctx context.Context, component string) {
	if panics == nil {
		return
	}
	panics.Add(ctx, 1, metric.WithAttributes(attribute.String("component", component)))
}

func MLPayloadSize(ctx context.Context, sizeBytes int64, flow string) {
	if mlPayloadSize == nil {
		return
	}
	mlPayloadSize.Record(ctx, sizeBytes, metric.WithAttributes(attribute.String("flow", flow)))
}

func MLPayloadMediaCount(ctx context.Context, count int64, flow string) {
	if mlPayloadMediaCount == nil {
		return
	}
	mlPayloadMediaCount.Record(ctx, count, metric.WithAttributes(attribute.String("flow", flow)))
}

func ObserveMLPresignDuration(ctx context.Context, seconds float64, flow string) {
	if mlPresignDuration == nil {
		return
	}
	mlPresignDuration.Record(ctx, seconds, metric.WithAttributes(attribute.String("flow", flow)))
}

func MLTextTaskItem(ctx context.Context, entityKind, field string) {
	if mlTextTaskItems == nil {
		return
	}
	mlTextTaskItems.Add(ctx, 1, metric.WithAttributes(
		attribute.String("entity_kind", entityKind),
		attribute.String("field", field),
	))
}

func MLTextResultApplied(ctx context.Context, entityKind, field string, censored bool) {
	if mlTextResultApplied == nil {
		return
	}
	mlTextResultApplied.Add(ctx, 1, metric.WithAttributes(
		attribute.String("entity_kind", entityKind),
		attribute.String("field", field),
		attribute.Bool("censored", censored),
	))
}

func MLTextResultSkipped(ctx context.Context, reason string) {
	if mlTextResultSkipped == nil {
		return
	}
	mlTextResultSkipped.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", reason)))
}
