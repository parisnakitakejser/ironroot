package irtop

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "irtop.yaml")
	if err := os.WriteFile(path, []byte(`server: https://ironroot.example
token: admin-token
ca_file: /tmp/root-ca.crt
insecure_skip_verify: true
refresh: 10s
default_view: telemetry
output: text
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "https://ironroot.example" {
		t.Fatalf("unexpected server %q", cfg.Server)
	}
	if cfg.Token != "admin-token" {
		t.Fatalf("unexpected token %q", cfg.Token)
	}
	if cfg.CAFile != "/tmp/root-ca.crt" {
		t.Fatalf("unexpected ca file %q", cfg.CAFile)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected insecure_skip_verify")
	}
	if cfg.Refresh != 10*time.Second {
		t.Fatalf("unexpected refresh %s", cfg.Refresh)
	}
	if cfg.DefaultView != "telemetry" {
		t.Fatalf("unexpected default view %q", cfg.DefaultView)
	}
	if cfg.Output != "text" {
		t.Fatalf("unexpected output %q", cfg.Output)
	}
}

func TestParseView(t *testing.T) {
	tests := map[string]View{
		"overview":     ViewOverview,
		"certificates": ViewCertificates,
		"certs":        ViewCertificates,
		"enrollments":  ViewEnrollments,
		"tokens":       ViewTokens,
		"ca-health":    ViewCAHealth,
		"security":     ViewSecurity,
		"telemetry":    ViewTelemetry,
		"audit-log":    ViewAuditLog,
		"server":       ViewServer,
		"unknown":      ViewOverview,
	}
	for input, want := range tests {
		if got := ParseView(input); got != want {
			t.Fatalf("ParseView(%q) = %v, want %v", input, got, want)
		}
	}
}
