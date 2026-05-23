package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ironroot/ironroot/internal/cli/admin"
	"github.com/ironroot/ironroot/internal/securitycheck"
	"github.com/ironroot/ironroot/internal/testutil"
)

func TestAdminBootstrapNonInteractiveGeneratesChecklist(t *testing.T) {
	dir := t.TempDir()
	checklist := filepath.Join(dir, "ironroot-security-checklist.md")
	cmd := admin.New()
	cmd.SetArgs([]string{"bootstrap", "--non-interactive", "--acknowledge-risk", "--output-checklist", checklist})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(checklist)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Offline Root CA checklist") || !strings.Contains(out.String(), "Bootstrap guide complete") {
		t.Fatal("expected checklist and completion output")
	}
}

func TestAdminSecurityCheckJSONAndFailOn(t *testing.T) {
	configPath := writeSecurityCheckConfig(t)
	cmd := admin.New()
	cmd.SetArgs([]string{"security-check", "--config", configPath, "--output", "json", "--fail-on", "critical"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var report securitycheck.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Checks) == 0 {
		t.Fatal("expected security checks")
	}

	cmd = admin.New()
	cmd.SetArgs([]string{"security-check", "--config", configPath, "--output", "json", "--fail-on", "high"})
	cmd.SetOut(&bytes.Buffer{})
	err := cmd.Execute()
	exitErr, ok := err.(admin.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for high failures, got %#v", err)
	}
}

func writeSecurityCheckConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fixture := testutil.NewTestCA(t)
	keyPath := filepath.Join(fixture.Dir, "intermediate.key")
	if err := os.Chmod(keyPath, 0o600); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "ironroot.db")
	configPath := filepath.Join(dir, "config.yaml")
	body := "server:\n  address: \":8443\"\n  tls:\n    cert_file: \"\"\n    key_file: \"\"\n" +
		"database:\n  driver: sqlite\n  dsn: \"file:" + dbPath + "?_foreign_keys=on\"\n" +
		"pki:\n  root_file: \"" + filepath.Join(fixture.Dir, "root-ca.crt") + "\"\n  chain_file: \"" + filepath.Join(fixture.Dir, "ca-chain.crt") + "\"\n  intermediate_cert_file: \"" + filepath.Join(fixture.Dir, "intermediate.crt") + "\"\n  intermediate_key_file: \"" + keyPath + "\"\n  default_lifetime: 2160h\n  renew_before: 720h\n" +
		"telemetry:\n  enabled: false\n  service_name: ironroot-server\n  service_version: 0.1.0\n  deployment_environment: test\n  otlp_endpoint: localhost:4317\n  otlp_protocol: grpc\n  sampling_ratio: 1\n" +
		"log:\n  level: info\n"
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath
}
