package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	apnsPushes metric.Int64Counter
	apnsDuration metric.Float64Histogram
	emails metric.Int64Counter
	streamConsumed metric.Int64Counter
	streamConsumeDuration metric.Float64Histogram
)

func Init() {
	m := otel.Meter("notification-service")

	apnsPushes, _ = m.Int64Counter("apns.pushes.total",
		metric.WithDescription("APNS push attempts by event type and result"),
	)
	apnsDuration, _ = m.Float64Histogram("apns.push.duration_seconds",
		metric.WithDescription("APNS push round-trip duration"),
		metric.WithUnit("s"),
	)
	emails, _ = m.Int64Counter("emails.sent.total",
		metric.WithDescription("Email send attempts by result (sent/failed/skipped)"),
	)
	streamConsumed, _ = m.Int64Counter("stream.messages.consumed.total",
		metric.WithDescription("Redis stream messages consumed"),
	)
	streamConsumeDuration, _ = m.Float64Histogram("stream.message.processing.duration_seconds",
		metric.WithDescription("Per-message processing duration"),
		metric.WithUnit("s"),
	)
}

func APNSPush(ctx context.Context, eventType, result string) {
	if apnsPushes == nil {
		return
	}
	apnsPushes.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("result", result),
	))
}

func ObserveAPNSDuration(ctx context.Context, seconds float64, result string) {
	if apnsDuration == nil {
		return
	}
	apnsDuration.Record(ctx, seconds, metric.WithAttributes(attribute.String("result", result)))
}

func Email(ctx context.Context, result string) {
	if emails == nil {
		return
	}
	emails.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
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
