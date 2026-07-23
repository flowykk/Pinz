package otel

import (
	"context"
	"database/sql"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RegisterDBPoolMetrics tags each observation with dbName so a service with
// multiple pools keeps them apart.
func RegisterDBPoolMetrics(meter metric.Meter, db *sql.DB, dbName string) error {
	if db == nil {
		return nil
	}

	open, err := meter.Int64ObservableGauge("db.connections.open",
		metric.WithDescription("Current number of open DB connections"),
	)
	if err != nil {
		return err
	}
	inUse, err := meter.Int64ObservableGauge("db.connections.in_use",
		metric.WithDescription("DB connections currently in use"),
	)
	if err != nil {
		return err
	}
	idle, err := meter.Int64ObservableGauge("db.connections.idle",
		metric.WithDescription("Idle DB connections"),
	)
	if err != nil {
		return err
	}
	waitCount, err := meter.Int64ObservableCounter("db.wait.count",
		metric.WithDescription("Cumulative number of connection waits"),
	)
	if err != nil {
		return err
	}
	waitDur, err := meter.Float64ObservableCounter("db.wait.duration_seconds",
		metric.WithDescription("Cumulative time spent waiting for a connection"),
	)
	if err != nil {
		return err
	}
	maxIdleClosed, err := meter.Int64ObservableCounter("db.closed.max_idle",
		metric.WithDescription("Connections closed due to SetMaxIdleConns"),
	)
	if err != nil {
		return err
	}
	maxLifetimeClosed, err := meter.Int64ObservableCounter("db.closed.max_lifetime",
		metric.WithDescription("Connections closed due to SetConnMaxLifetime"),
	)
	if err != nil {
		return err
	}

	attrs := metric.WithAttributes(attribute.String("db", dbName))

	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		s := db.Stats()
		o.ObserveInt64(open, int64(s.OpenConnections), attrs)
		o.ObserveInt64(inUse, int64(s.InUse), attrs)
		o.ObserveInt64(idle, int64(s.Idle), attrs)
		o.ObserveInt64(waitCount, s.WaitCount, attrs)
		o.ObserveFloat64(waitDur, s.WaitDuration.Seconds(), attrs)
		o.ObserveInt64(maxIdleClosed, s.MaxIdleClosed, attrs)
		o.ObserveInt64(maxLifetimeClosed, s.MaxLifetimeClosed, attrs)
		return nil
	}, open, inUse, idle, waitCount, waitDur, maxIdleClosed, maxLifetimeClosed)
	return err
}
