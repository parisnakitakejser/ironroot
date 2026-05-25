package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(t.TempDir() + "/missing.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("driver = %s", cfg.Database.Driver)
	}
	if cfg.PKI.DefaultLifetime == 0 || cfg.PKI.RenewBefore == 0 {
		t.Fatal("expected certificate lifetime defaults")
	}
}

func TestLoadRBACConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`rbac:
  enabled: true
  mode: file
  paths:
    - config/rbac/*.yaml
    - config/rbac/*.yml
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.RBAC.Enabled || cfg.RBAC.Mode != "file" || len(cfg.RBAC.Paths) != 2 {
		t.Fatalf("unexpected RBAC config: %+v", cfg.RBAC)
	}
}

func TestLoadTelemetryEnvironment(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector.internal:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	t.Setenv("OTEL_SERVICE_NAME", "ironroot-ci")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment=test")
	t.Setenv("OTEL_TRACES_SAMPLER", "traceidratio")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.25")

	cfg, err := Load(t.TempDir() + "/missing.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telemetry.Exporter.Endpoint != "collector.internal:4317" || cfg.Telemetry.OTLPEndpoint != "collector.internal:4317" {
		t.Fatalf("unexpected OTLP endpoint config: %+v", cfg.Telemetry)
	}
	if cfg.Telemetry.ServiceName != "ironroot-ci" {
		t.Fatalf("service name = %q", cfg.Telemetry.ServiceName)
	}
	if cfg.Telemetry.DeploymentEnvironment != "test" {
		t.Fatalf("deployment environment = %q", cfg.Telemetry.DeploymentEnvironment)
	}
	if cfg.Telemetry.Sampling.Ratio != 0.25 || cfg.Telemetry.SamplingRatio != 0.25 {
		t.Fatalf("sampling config = %+v", cfg.Telemetry)
	}
}
