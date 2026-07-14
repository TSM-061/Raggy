package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.41.0"
)

type Config struct {
	CollectorEndpoint string `env:"COLLECTOR_ENDPOINT,required"`
	DisableTLS        bool   `env:"DISABLE_TLS" envDefault:"true"`
}

func initTextMapPropagator() {
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
}

func InitTracerProvider(ctx context.Context, serviceName string, cfg *Config) (*trace.TracerProvider, error) {
	exporterOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.CollectorEndpoint),
	}
	if cfg.DisableTLS {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.DeploymentEnvironmentNameDevelopment,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter, trace.WithBatchTimeout(50*time.Millisecond)),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // Capture all in development
	)

	initTextMapPropagator()

	otel.SetTracerProvider(tp)

	return tp, nil
}
