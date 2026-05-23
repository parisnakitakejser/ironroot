package securitycheck

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/ironroot/ironroot/internal/telemetry"
)

type Runner struct {
	Checks []Check
}

func DefaultRunner() Runner {
	return Runner{Checks: []Check{
		HostCheck{},
		EnvironmentSecretsCheck{},
		FilePermissionCheck{IDValue: "host.config_permissions", Title: "Config file permissions are restricted", CategoryValue: "host", PathKind: "config"},
		FilePermissionCheck{IDValue: "host.ca_permissions", Title: "CA private key permissions are restricted", CategoryValue: "host", PathKind: "ca_key"},
		FilePermissionCheck{IDValue: "host.database_permissions", Title: "SQLite database permissions are restricted", CategoryValue: "database", PathKind: "database"},
		RootPrivateKeyAbsentCheck{},
		IntermediateKeyCheck{},
		CAChainCheck{},
		CertificateLifetimeCheck{},
		DatabaseReachableCheck{},
		SQLiteKubernetesCheck{},
		APITLSCheck{},
		APIBindCheck{},
		TokenTTLCheck{},
		AuditCheck{},
		TelemetryCheck{},
		ContainerCheck{},
		KubernetesCheck{},
	}}
}

func (r Runner) Run(ctx context.Context, target Target) Report {
	if target.Now.IsZero() {
		target.Now = time.Now().UTC()
	}
	results := make([]Result, 0, len(r.Checks))
	for _, check := range r.Checks {
		category := check.Category()
		ctx, span := otel.Tracer("ironroot-securitycheck").Start(ctx, "security-check "+category)
		result := check.Run(ctx, target)
		span.SetAttributes(
			attribute.String("check.id", result.ID),
			attribute.String("check.category", result.Category),
			attribute.String("check.severity", string(result.Severity)),
			attribute.String("check.status", string(result.Status)),
		)
		span.End()
		telemetry.Instruments().SecurityCheckResults.Add(ctx, 1, metric.WithAttributes(
			attribute.String("severity", string(result.Severity)),
			attribute.String("status", string(result.Status)),
			attribute.String("category", result.Category),
		))
		results = append(results, result)
	}
	SortResults(results)
	return Report{Summary: Summarize(results), Checks: results}
}
