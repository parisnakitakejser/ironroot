package telemetry

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("github.com/ironroot/ironroot")
	once  sync.Once
	m     Metrics
)

type Metrics struct {
	APIRequests             metric.Int64Counter
	APIRequestDuration      metric.Float64Histogram
	CertificatesIssued      metric.Int64Counter
	CertificatesRenewed     metric.Int64Counter
	CertificatesRevoked     metric.Int64Counter
	Enrollments             metric.Int64Counter
	EnrollmentFailures      metric.Int64Counter
	TokenValidationFailures metric.Int64Counter
	CLICommandDuration      metric.Float64Histogram
	CLICommandErrors        metric.Int64Counter
	SecurityCheckResults    metric.Int64Counter
}

func Instruments() Metrics {
	once.Do(func() {
		m.APIRequests, _ = meter.Int64Counter("pki_api_requests_total")
		m.APIRequestDuration, _ = meter.Float64Histogram("pki_api_request_duration_seconds")
		m.CertificatesIssued, _ = meter.Int64Counter("pki_certificates_issued_total")
		m.CertificatesRenewed, _ = meter.Int64Counter("pki_certificates_renewed_total")
		m.CertificatesRevoked, _ = meter.Int64Counter("pki_certificates_revoked_total")
		m.Enrollments, _ = meter.Int64Counter("pki_enrollments_total")
		m.EnrollmentFailures, _ = meter.Int64Counter("pki_enrollment_failures_total")
		m.TokenValidationFailures, _ = meter.Int64Counter("pki_bootstrap_token_validation_failures_total")
		m.CLICommandDuration, _ = meter.Float64Histogram("pki_cli_command_duration_seconds")
		m.CLICommandErrors, _ = meter.Int64Counter("pki_cli_command_errors_total")
		m.SecurityCheckResults, _ = meter.Int64Counter("pki_security_check_results_total")
	})
	return m
}

func RecordCommand(ctx context.Context, name string, started time.Time, err error) {
	metrics := Instruments()
	metrics.CLICommandDuration.Record(ctx, time.Since(started).Seconds())
	if err != nil {
		metrics.CLICommandErrors.Add(ctx, 1)
	}
}
