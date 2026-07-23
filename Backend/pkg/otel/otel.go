// Package otel provides a reusable OpenTelemetry initialisation helper for
// Pinz backend services. All configuration is driven by standard OTel
// environment variables:
// OTEL_EXPORTER_OTLP_ENDPOINT – collector gRPC endpoint (default: localhost:4317)
// OTEL_SERVICE_NAME – falls back to the serviceName argument
// OTEL_RESOURCE_ATTRIBUTES – extra key=value pairs merged into the resource
// OTEL_TRACES_SAMPLER – sampler type (default: parentbased_always_on)
package otel

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Providers holds all initialised OTel provider references so callers can
// perform a coordinated graceful shutdown.
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// Init initialises TracerProvider, MeterProvider and LoggerProvider with
// OTLP/gRPC exporters and sets them as the global providers. W3C TraceContext
// + Baggage propagation is also configured globally.
// Call Providers.Shutdown in a deferred function inside main to flush
// in-flight telemetry on process exit.
func Init(ctx context.Context, serviceName, version string) (*Providers, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(version),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	// ── Tracing ──────────────────────────────────────────────────────────────

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// ── Metrics ──────────────────────────────────────────────────────────────

	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			metricExporter,
			sdkmetric.WithInterval(5*time.Second),
		)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	// Force at least one metric so the first export is non-empty (helps verify pipeline).
	meter := otel.Meter(serviceName)
	if c, err := meter.Int64Counter("pinz.service.started", metric.WithDescription("1 if OTel was initialized")); err == nil {
		c.Add(ctx, 1)
	}

	// ── Logs ─────────────────────────────────────────────────────────────────

	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil, fmt.Errorf("otlp log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	return &Providers{
		TracerProvider: tp,
		MeterProvider: mp,
		LoggerProvider: lp,
	}, nil
}

// Shutdown flushes all pending telemetry and terminates all providers.
// Use a separate context with a reasonable timeout (e.g. 5 s) to avoid
// blocking indefinitely on process exit.
func (p *Providers) Shutdown(ctx context.Context) {
	_ = p.LoggerProvider.Shutdown(ctx)
	_ = p.MeterProvider.Shutdown(ctx)
	_ = p.TracerProvider.Shutdown(ctx)
}
