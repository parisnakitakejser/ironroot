package telemetry

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/parisnakitakejser/ironroot/internal/config"
)

func TestConfigureDisabled(t *testing.T) {
	shutdown, err := Configure(context.Background(), config.TelemetryConfig{Enabled: false}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSanitizeLabel(t *testing.T) {
	got := SanitizeLabel("request-cert --token secret:value")
	if strings.Contains(got, " ") || strings.Contains(got, ":") {
		t.Fatalf("expected unsafe label characters to be replaced, got %q", got)
	}
	if SanitizeLabel("") != "unknown" {
		t.Fatal("expected empty labels to become unknown")
	}
}

func TestRecordCommandMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
	})

	RecordCommand(context.Background(), "enroll", time.Now().Add(-time.Second), errors.New("boom"))

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	assertMetricExists(t, rm, "pki_cli_command_duration_seconds")
	assertMetricExists(t, rm, "pki_cli_command_errors_total")
	assertMetricExists(t, rm, "pki_cli_command_results_total")
}

func TestLoggerIncludesTraceIDs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "debug")
	provider := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
	})

	ctx, span := otel.Tracer("test").Start(context.Background(), "log-test")
	logger.InfoContext(ctx, "hello", "component", "test")
	span.End()

	out := buf.String()
	if !strings.Contains(out, "trace_id") || !strings.Contains(out, "span_id") {
		t.Fatalf("expected trace correlation fields in log output: %s", out)
	}
}

func assertMetricExists(t *testing.T, rm metricdata.ResourceMetrics, name string) {
	t.Helper()
	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name == name {
				return
			}
		}
	}
	t.Fatalf("metric %q was not collected", name)
}
