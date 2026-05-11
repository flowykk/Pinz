package metrics

import (
	"context"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	jwtFailures metric.Int64Counter
	wsConnects metric.Int64Counter
	wsDisconnects metric.Int64Counter
	wsActiveGauge metric.Int64ObservableGauge

	wsActiveMu sync.Mutex
	wsActive = map[string]*int64{}
)

func Init() {
	m := otel.Meter("api-gateway")

	jwtFailures, _ = m.Int64Counter("jwt.failures.total",
		metric.WithDescription("JWT validation failures by reason"),
	)
	wsConnects, _ = m.Int64Counter("ws.connects.total",
		metric.WithDescription("WebSocket connection attempts by endpoint and result"),
	)
	wsDisconnects, _ = m.Int64Counter("ws.disconnects.total",
		metric.WithDescription("WebSocket disconnects by endpoint and reason"),
	)
	wsActiveGauge, _ = m.Int64ObservableGauge("ws.connections.active",
		metric.WithDescription("Active WebSocket connections by endpoint"),
	)
	if wsActiveGauge != nil {
		_, _ = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
			wsActiveMu.Lock()
			defer wsActiveMu.Unlock()
			for endpoint, n := range wsActive {
				o.ObserveInt64(wsActiveGauge, atomic.LoadInt64(n), metric.WithAttributes(attribute.String("endpoint", endpoint)))
			}
			return nil
		}, wsActiveGauge)
	}
}

func JWTFailure(ctx context.Context, reason string) {
	if jwtFailures == nil {
		return
	}
	jwtFailures.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", reason)))
}

func WSConnect(ctx context.Context, endpoint, result string) {
	if wsConnects == nil {
		return
	}
	wsConnects.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("result", result),
	))
}

func WSDisconnect(ctx context.Context, endpoint, reason string) {
	if wsDisconnects == nil {
		return
	}
	wsDisconnects.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("reason", reason),
	))
}

func IncWSActive(endpoint string) {
	wsActiveMu.Lock()
	defer wsActiveMu.Unlock()
	v, ok := wsActive[endpoint]
	if !ok {
		var zero int64
		v = &zero
		wsActive[endpoint] = v
	}
	atomic.AddInt64(v, 1)
}

func DecWSActive(endpoint string) {
	wsActiveMu.Lock()
	defer wsActiveMu.Unlock()
	if v, ok := wsActive[endpoint]; ok {
		atomic.AddInt64(v, -1)
	}
}
