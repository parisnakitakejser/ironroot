package telemetry

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"

	"github.com/ironroot/ironroot/internal/config"
)

type Shutdown func(context.Context) error

func Configure(ctx context.Context, cfg config.TelemetryConfig, fallbackName string) (Shutdown, error) {
	ctx, span := StartSpan(ctx, "telemetry.configure")
	defer span.End()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if cfg.Sampling.Ratio != 0 {
		cfg.SamplingRatio = cfg.Sampling.Ratio
	}
	if cfg.Exporter.Endpoint != "" {
		cfg.OTLPEndpoint = cfg.Exporter.Endpoint
	}
	if cfg.Exporter.Protocol != "" {
		cfg.OTLPProtocol = cfg.Exporter.Protocol
	}
	if cfg.Exporter.Insecure {
		cfg.Exporter.Insecure = true
	}
	if !cfg.Enabled && !cfg.Prometheus.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	name := cfg.ServiceName
	if name == "" {
		name = fallbackName
	}
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(name),
		semconv.ServiceVersion(cfg.ServiceVersion),
		semconv.DeploymentEnvironmentName(cfg.DeploymentEnvironment),
	))
	if err != nil {
		return nil, err
	}
	var shutdowns []Shutdown
	if cfg.Enabled && cfg.Traces.Enabled {
		started := time.Now()
		var traceExporter sdktrace.SpanExporter
		switch strings.ToLower(cfg.OTLPProtocol) {
		case "http", "http/protobuf":
			opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.OTLPEndpoint)}
			if cfg.Exporter.Insecure {
				opts = append(opts, otlptracehttp.WithInsecure())
			}
			traceExporter, err = otlptracehttp.New(ctx, opts...)
		default:
			opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint)}
			if cfg.Exporter.Insecure {
				opts = append(opts, otlptracegrpc.WithInsecure())
			}
			traceExporter, err = otlptracegrpc.New(ctx, opts...)
		}
		RecordExporter(ctx, "traces.init", started, err)
		if err != nil {
			return nil, err
		}
		sampler := sdktrace.TraceIDRatioBased(cfg.SamplingRatio)
		tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(res), sdktrace.WithSampler(sampler))
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, tp.Shutdown)
	}

	var readers []metric.Reader
	if cfg.Enabled && cfg.Metrics.Enabled {
		started := time.Now()
		var metricExporter metric.Exporter
		switch strings.ToLower(cfg.OTLPProtocol) {
		case "http", "http/protobuf":
			opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.OTLPEndpoint)}
			if cfg.Exporter.Insecure {
				opts = append(opts, otlpmetrichttp.WithInsecure())
			}
			metricExporter, err = otlpmetrichttp.New(ctx, opts...)
		default:
			opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint)}
			if cfg.Exporter.Insecure {
				opts = append(opts, otlpmetricgrpc.WithInsecure())
			}
			metricExporter, err = otlpmetricgrpc.New(ctx, opts...)
		}
		RecordExporter(ctx, "metrics.init", started, err)
		if err != nil {
			return nil, err
		}
		readers = append(readers, metric.NewPeriodicReader(metricExporter))
	}
	if cfg.Prometheus.Enabled {
		exporter, err := prometheus.New()
		if err != nil {
			return nil, err
		}
		readers = append(readers, exporter)
	}
	if len(readers) > 0 {
		opts := []metric.Option{metric.WithResource(res)}
		for _, reader := range readers {
			opts = append(opts, metric.WithReader(reader))
		}
		mp := metric.NewMeterProvider(opts...)
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, mp.Shutdown)
	}

	return func(ctx context.Context) error {
		var errs []error
		for _, shutdown := range shutdowns {
			errs = append(errs, shutdown(ctx))
		}
		return errors.Join(errs...)
	}, nil
}
