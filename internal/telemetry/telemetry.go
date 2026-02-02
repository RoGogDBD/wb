// Package telemetry содержит инициализацию трассировки и метрик.
package telemetry

import (
	"context"
	"errors"
	"net/http"

	"github.com/RoGogDBD/wb/internal/config"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Providers содержит активные компоненты телеметрии.
type Providers struct {
	MetricsHandler http.Handler
	shutdown       func(context.Context) error
}

// Shutdown корректно завершает все провайдеры.
func (p *Providers) Shutdown(ctx context.Context) error {
	if p == nil || p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}

// Init инициализирует трассировку и метрики.
func Init(ctx context.Context, cfg config.TelemetryConfig) (*Providers, error) {
	if !cfg.TracesEnabled && !cfg.MetricsEnabled {
		return &Providers{}, nil
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("deployment.environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	var shutdowns []func(context.Context) error
	var metricsHandler http.Handler

	if cfg.TracesEnabled {
		options := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		}
		if cfg.OTLPInsecure {
			options = append(options, otlptracehttp.WithInsecure())
		}
		traceExporter, err := otlptracehttp.New(ctx, options...)
		if err != nil {
			return nil, err
		}
		sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TraceSampleRatio))
		traceProvider := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sampler),
		)
		otel.SetTracerProvider(traceProvider)
		shutdowns = append(shutdowns, traceProvider.Shutdown)
	}

	if cfg.MetricsEnabled {
		registry := prom.NewRegistry()
		metricExporter, err := otelprom.New(otelprom.WithRegisterer(registry))
		if err != nil {
			return nil, err
		}
		metricProvider := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(metricExporter),
		)
		otel.SetMeterProvider(metricProvider)
		metricsHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		shutdowns = append(shutdowns, metricProvider.Shutdown)
	}

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return &Providers{
		MetricsHandler: metricsHandler,
		shutdown: func(ctx context.Context) error {
			var joined error
			for _, shutdown := range shutdowns {
				if err := shutdown(ctx); err != nil {
					joined = errors.Join(joined, err)
				}
			}
			return joined
		},
	}, nil
}
