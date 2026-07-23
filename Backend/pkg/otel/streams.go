package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type StreamPending struct {
	Count int64
	MaxIdle time.Duration
}

// StreamQueryer keeps pkg/otel free of a go-redis dependency.
type StreamQueryer interface {
	XLen(ctx context.Context, stream string) (int64, error)
	XPending(ctx context.Context, stream, group string) (StreamPending, error)
}

type StreamSpec struct {
	Stream string
	Groups []string
}

func RegisterStreamMetrics(meter metric.Meter, q StreamQueryer, specs []StreamSpec) error {
	if q == nil || len(specs) == 0 {
		return nil
	}

	length, err := meter.Int64ObservableGauge("stream.length",
		metric.WithDescription("Redis Stream length (XLEN)"),
	)
	if err != nil {
		return err
	}
	pending, err := meter.Int64ObservableGauge("stream.pending",
		metric.WithDescription("Pending messages per consumer group (XPENDING count)"),
	)
	if err != nil {
		return err
	}
	pendingIdle, err := meter.Float64ObservableGauge("stream.pending.idle_max_seconds",
		metric.WithDescription("Max idle time of pending messages in a group"),
	)
	if err != nil {
		return err
	}

	_, err = meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		for _, spec := range specs {
			streamAttr := attribute.String("stream", spec.Stream)

			sCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			if n, err := q.XLen(sCtx, spec.Stream); err == nil {
				o.ObserveInt64(length, n, metric.WithAttributes(streamAttr))
			}
			cancel()

			for _, group := range spec.Groups {
				gCtx, gCancel := context.WithTimeout(ctx, 500*time.Millisecond)
				if p, err := q.XPending(gCtx, spec.Stream, group); err == nil {
					attrs := metric.WithAttributes(streamAttr, attribute.String("group", group))
					o.ObserveInt64(pending, p.Count, attrs)
					o.ObserveFloat64(pendingIdle, p.MaxIdle.Seconds(), attrs)
				}
				gCancel()
			}
		}
		return nil
	}, length, pending, pendingIdle)
	return err
}
