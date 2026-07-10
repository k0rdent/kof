/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package telemetry initialises the OpenTelemetry SDK for the kof-operator
// process.  Configuration is read entirely from the standard OTel environment
// variables so that the same binary works across different deployment
// topologies without recompilation:
//
//   - OTEL_EXPORTER_OTLP_ENDPOINT  – collector address (optional; telemetry is disabled when unset)
//   - OTEL_EXPORTER_OTLP_PROTOCOL  – transport protocol (defaults to "grpc")
//   - OTEL_SERVICE_NAME             – service name reported in every span
//   - OTEL_RESOURCE_ATTRIBUTES      – extra resource attributes (k=v,k=v)
//
// If OTEL_EXPORTER_OTLP_ENDPOINT is not set, Setup is a no-op and returns a
// no-op (non-nil) shutdown function.
package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// ShutdownFunc flushes and stops the TracerProvider. It should be called on
// process exit. Setup always returns a non-nil ShutdownFunc.
type ShutdownFunc func(context.Context) error

// Setup initialises the global OTel TracerProvider and TextMapPropagator.
// It returns a ShutdownFunc that must be called before the process exits to
// flush pending spans.
//
// If OTEL_EXPORTER_OTLP_ENDPOINT is empty, Setup returns a no-op shutdown
// function and a nil error – telemetry is simply disabled.
func Setup(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(_ context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP gRPC trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		// Non-fatal: proceed with a minimal resource that still carries
		// service.name so traces remain attributable.
		res, _ = resource.Merge(
			resource.Default(),
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
			),
		)
		if res == nil {
			res = resource.Default()
		}
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// NewTransport wraps base with an OTel client-side transport.
// Spans are named "METHOD host/path" for clear identification in traces.
// If base is nil, http.DefaultTransport is used.
func NewTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Host + r.URL.Path
		}),
	)
}

// StartReconcileSpan starts a new OTel span for a controller reconcile loop.
// The span is named "Reconcile kind name/namespace".
// The returned context carries the span; callers must call end() when done.
//
//	ctx, end := telemetry.StartReconcileSpan(ctx, "PromxyServerGroup", req)
//	defer end()
func StartReconcileSpan(ctx context.Context, kind string, name, namespace string) (context.Context, func()) {
	tracer := otel.Tracer("kof-operator/controller")
	spanName := "Reconcile " + kind + " " + namespace + "/" + name
	ctx, span := tracer.Start(ctx, spanName)
	return ctx, func() { span.End() }
}
