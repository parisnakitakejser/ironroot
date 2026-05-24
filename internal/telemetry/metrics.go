package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	APIRequests             metric.Int64Counter
	APIRequestFailures      metric.Int64Counter
	APIRequestDuration      metric.Float64Histogram
	CertificatesIssued      metric.Int64Counter
	CertificatesRenewed     metric.Int64Counter
	CertificatesRevoked     metric.Int64Counter
	ActiveCertificates      metric.Int64UpDownCounter
	ExpiringCertificates    metric.Int64UpDownCounter
	Enrollments             metric.Int64Counter
	EnrollmentFailures      metric.Int64Counter
	TokenValidationFailures metric.Int64Counter
	BootstrapRuns           metric.Int64Counter
	BootstrapFailures       metric.Int64Counter
	DatabaseQueryDuration   metric.Float64Histogram
	DatabaseErrors          metric.Int64Counter
	OTelExportFailures      metric.Int64Counter
	OTelExportDuration      metric.Float64Histogram
	CLICommandDuration      metric.Float64Histogram
	CLICommandErrors        metric.Int64Counter
	CLICommandResults       metric.Int64Counter
	SecurityCheckResults    metric.Int64Counter
}

func Instruments() Metrics {
	meter := otel.Meter("github.com/parisnakitakejser/ironroot")
	m := Metrics{}
	m.APIRequests, _ = meter.Int64Counter("pki_api_requests_total")
	m.APIRequestFailures, _ = meter.Int64Counter("pki_api_request_failures_total")
	m.APIRequestDuration, _ = meter.Float64Histogram("pki_api_request_duration_seconds")
	m.CertificatesIssued, _ = meter.Int64Counter("pki_certificates_issued_total")
	m.CertificatesRenewed, _ = meter.Int64Counter("pki_certificates_renewed_total")
	m.CertificatesRevoked, _ = meter.Int64Counter("pki_certificates_revoked_total")
	m.ActiveCertificates, _ = meter.Int64UpDownCounter("pki_active_certificates_total")
	m.ExpiringCertificates, _ = meter.Int64UpDownCounter("pki_expiring_certificates_total")
	m.Enrollments, _ = meter.Int64Counter("pki_enrollments_total")
	m.EnrollmentFailures, _ = meter.Int64Counter("pki_enrollment_failures_total")
	m.TokenValidationFailures, _ = meter.Int64Counter("pki_bootstrap_token_validation_failures_total")
	m.BootstrapRuns, _ = meter.Int64Counter("pki_bootstrap_runs_total")
	m.BootstrapFailures, _ = meter.Int64Counter("pki_bootstrap_failures_total")
	m.DatabaseQueryDuration, _ = meter.Float64Histogram("pki_database_query_duration_seconds")
	m.DatabaseErrors, _ = meter.Int64Counter("pki_database_errors_total")
	m.OTelExportFailures, _ = meter.Int64Counter("pki_otel_export_failures_total")
	m.OTelExportDuration, _ = meter.Float64Histogram("pki_otel_export_duration_seconds")
	m.CLICommandDuration, _ = meter.Float64Histogram("pki_cli_command_duration_seconds")
	m.CLICommandErrors, _ = meter.Int64Counter("pki_cli_command_errors_total")
	m.CLICommandResults, _ = meter.Int64Counter("pki_cli_command_results_total")
	m.SecurityCheckResults, _ = meter.Int64Counter("pki_security_check_results_total")
	return m
}

func RecordCommand(ctx context.Context, name string, started time.Time, err error) {
	metrics := Instruments()
	status := "success"
	if err != nil {
		status = "error"
	}
	attrs := metric.WithAttributes(attribute.String("command", SanitizeLabel(name)), attribute.String("status", status))
	metrics.CLICommandDuration.Record(ctx, time.Since(started).Seconds(), attrs)
	metrics.CLICommandResults.Add(ctx, 1, attrs)
	if err != nil {
		metrics.CLICommandErrors.Add(ctx, 1, attrs)
	}
}

func SanitizeLabel(value string) string {
	if value == "" {
		return "unknown"
	}
	out := make([]rune, 0, len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/' || r == '.' {
			out = append(out, r)
			continue
		}
		out = append(out, '_')
	}
	if len(out) > 80 {
		out = out[:80]
	}
	return string(out)
}
