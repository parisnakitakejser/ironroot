package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return otel.Tracer("github.com/parisnakitakejser/ironroot").Start(ctx, name, trace.WithAttributes(attrs...))
}

func EndSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func RecordDatabase(ctx context.Context, operation string, started time.Time, err error) {
	attrs := metric.WithAttributes(attribute.String("operation", SanitizeLabel(operation)))
	metrics := Instruments()
	metrics.DatabaseQueryDuration.Record(ctx, time.Since(started).Seconds(), attrs)
	if err != nil {
		metrics.DatabaseErrors.Add(ctx, 1, attrs)
	}
}

func RecordExporter(ctx context.Context, signal string, started time.Time, err error) {
	attrs := metric.WithAttributes(attribute.String("signal", SanitizeLabel(signal)))
	metrics := Instruments()
	metrics.OTelExportDuration.Record(ctx, time.Since(started).Seconds(), attrs)
	if err != nil {
		metrics.OTelExportFailures.Add(ctx, 1, attrs)
	}
}
