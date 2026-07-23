package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	geocoding metric.Int64Counter
	geocodingDuration metric.Float64Histogram
	streamConsumed metric.Int64Counter
	streamConsumeDuration metric.Float64Histogram
)

func Init() {
	m := otel.Meter("statistics-service")

	geocoding, _ = m.Int64Counter("geocoding.requests.total",
		metric.WithDescription("Reverse geocoding requests by result"),
	)
	geocodingDuration, _ = m.Float64Histogram("geocoding.duration_seconds",
		metric.WithDescription("Reverse geocoding latency"),
		metric.WithUnit("s"),
	)
	streamConsumed, _ = m.Int64Counter("stream.messages.consumed.total",
		metric.WithDescription("Redis stream messages consumed"),
	)
	streamConsumeDuration, _ = m.Float64Histogram("stream.message.processing.duration_seconds",
		metric.WithDescription("Per-message processing duration"),
		metric.WithUnit("s"),
	)
}

func Geocoding(ctx context.Context, result string) {
	if geocoding == nil {
		return
	}
	geocoding.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func ObserveGeocodingDuration(ctx context.Context, seconds float64, result string) {
	if geocodingDuration == nil {
		return
	}
	geocodingDuration.Record(ctx, seconds, metric.WithAttributes(attribute.String("result", result)))
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
