// Package observability wires optional OpenTelemetry distributed tracing.
//
// Tracing is opt-in (TRACING_ENABLED=true) and always degrades gracefully:
// if the collector (Jaeger) is unreachable, exports fail in the background and
// the application keeps serving requests normally. Nothing here is on the hot
// path of a failed export.
package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// ShutdownFunc flushes and stops the tracer provider. Always non-nil after
// InitTracer returns (a no-op when tracing is disabled), so it is safe to defer.
type ShutdownFunc func(context.Context) error

func noopShutdown(context.Context) error { return nil }

// InitTracer configures the global tracer provider from environment variables:
//
//	TRACING_ENABLED            "true"/"1" to enable (default: disabled)
//	OTEL_EXPORTER_OTLP_ENDPOINT  collector host:port (default: localhost:4318)
//	OTEL_SERVICE_NAME            service name in Jaeger (default: l-xor-api)
//
// When disabled it returns a no-op shutdown and no error, so callers can wire it
// unconditionally. An exporter/connection problem is never fatal.
func InitTracer(ctx context.Context, log *slog.Logger) (ShutdownFunc, error) {
	if !isEnabled(os.Getenv("TRACING_ENABLED")) {
		log.Info("tracing disabled (set TRACING_ENABLED=true to enable)")
		return noopShutdown, nil
	}

	endpoint := normalizeEndpoint(getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318"))
	serviceName := getEnv("OTEL_SERVICE_NAME", "l-xor-api")

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // local dev: plain HTTP, no TLS
	)
	if err != nil {
		return noopShutdown, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		// Fall back to a minimal resource rather than failing startup.
		res = resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Keep background export failures (e.g. Jaeger down) out of the main logs at
	// error level — log them quietly so the app stays calm without a collector.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Debug("otel export error", "err", err)
	}))

	log.Info("tracing enabled", "service", serviceName, "endpoint", endpoint)

	return func(shutdownCtx context.Context) error {
		ctx, cancel := context.WithTimeout(shutdownCtx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}, nil
}

func isEnabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

// normalizeEndpoint strips a scheme/trailing slash because otlptracehttp expects
// a bare host:port (e.g. "localhost:4318"), not a full URL.
func normalizeEndpoint(ep string) string {
	ep = strings.TrimSpace(ep)
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	return strings.TrimRight(ep, "/")
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
