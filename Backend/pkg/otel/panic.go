package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// PanicCounter's zero value is a safe noop (Inc no-ops on a nil counter).
type PanicCounter struct {
	c metric.Int64Counter
}

func NewPanicCounter(meter metric.Meter) PanicCounter {
	c, _ := meter.Int64Counter("panics.total",
		metric.WithDescription("Number of recovered panics by component"),
	)
	return PanicCounter{c: c}
}

func (p PanicCounter) Inc(ctx context.Context, component string) {
	if p.c == nil {
		return
	}
	p.c.Add(ctx, 1, metric.WithAttributes(attribute.String("component", component)))
}
