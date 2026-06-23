package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func tracingEnabled() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""
}

func initTracing(ctx context.Context, cfg config, hostname string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if !tracingEnabled() {
		return noop, nil
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return noop, err
	}

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceInstanceID(hostname),
		attribute.String("zone", cfg.Zone),
		attribute.String("host", hostname),
		attribute.String("env", "poc"),
	))
	if err != nil {
		return noop, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(2*time.Second)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func instrumentHandler(h http.Handler) http.Handler {
	if !tracingEnabled() {
		return h
	}
	return otelhttp.NewHandler(h, "http.server")
}
