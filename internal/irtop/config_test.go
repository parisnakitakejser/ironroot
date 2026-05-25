package irtop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "irtop.yaml")
	if err := os.WriteFile(path, []byte(`default_profile: prod
profiles:
  prod:
    endpoint: https://ironroot.example
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

func TestLoadConfigDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".ironroot")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(`profiles:
  local:
    endpoint: http://localhost:8443
    refresh: 5s
    default_view: overview
    output: tui
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "http://localhost:8443" {
		t.Fatalf("unexpected server %q", cfg.Server)
	}
}

func TestLoadConfigExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config"), []byte(`profiles:
  prod:
    endpoint: https://ironroot.example
    ca_file: ~/root-ca.crt
    refresh: 5s
    default_view: overview
    output: tui
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig("~/config")
	if err != nil {
		t.Fatal(err)
	}
	wantCAFile := filepath.Join(home, "root-ca.crt")
	if cfg.CAFile != wantCAFile {
		t.Fatalf("CAFile = %q, want %q", cfg.CAFile, wantCAFile)
	}
}

func TestLoadProfilesWithMultipleProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(`default_profile: prod
profiles:
  local:
    server: http://localhost:8443
    refresh: 5s
    default_view: overview
    output: tui
  prod:
    server: https://ironroot.example.com:8443
    ca_file: ~/root-ca.crt
    refresh: 10s
    default_view: security
    output: tui
`), 0o600); err != nil {
		t.Fatal(err)
	}

	profiles, err := LoadProfiles(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles.Profiles))
	}
	if profiles.Profiles[profiles.Active].Name != "prod" {
		t.Fatalf("active profile = %q, want prod", profiles.Profiles[profiles.Active].Name)
	}
	if profiles.ActiveConfig().Server != "https://ironroot.example.com:8443" {
		t.Fatalf("unexpected active server %q", profiles.ActiveConfig().Server)
	}
}

func TestProfileSetSelect(t *testing.T) {
	profiles := ProfileSet{Profiles: []Profile{
		{Name: "local", Config: Config{Server: "http://localhost:8443"}},
		{Name: "prod", Config: Config{Server: "https://ironroot.example.com:8443"}},
	}}
	if err := profiles.Select("prod"); err != nil {
		t.Fatal(err)
	}
	if profiles.Active != 1 {
		t.Fatalf("active = %d, want 1", profiles.Active)
	}
	if err := profiles.Select("missing"); err == nil || !strings.Contains(err.Error(), `profile "missing" was not found`) {
		t.Fatalf("expected missing profile error, got %v", err)
	}
}

func TestLoadProfilesSupportsEndpointAlias(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(`profiles:
  local:
    endpoint: http://localhost:8443
    refresh: 5s
    default_view: overview
    output: tui
`), 0o600); err != nil {
		t.Fatal(err)
	}

	profiles, err := LoadProfiles(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := profiles.ActiveConfig().Server; got != "http://localhost:8443" {
		t.Fatalf("server = %q, want endpoint alias value", got)
	}
}

func TestLoadProfilesRejectsDuplicateProfileNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(`profiles:
  local:
    server: http://localhost:8443
    refresh: 5s
    default_view: overview
    output: tui
  local:
    server: http://localhost:9443
    refresh: 5s
    default_view: overview
    output: tui
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProfiles(path)
	if err == nil || !strings.Contains(err.Error(), `duplicate profile name "local"`) {
		t.Fatalf("expected duplicate profile name error, got %v", err)
	}
}

func TestLoadProfilesRejectsInvalidProfileShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(`profiles:
  local: http://localhost:8443
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProfiles(path)
	if err == nil || !strings.Contains(err.Error(), `profile "local" must be a YAML mapping`) {
		t.Fatalf("expected invalid profile shape error, got %v", err)
	}
}

func TestLoadProfilesRejectsConflictingServerAndEndpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(`profiles:
  local:
    server: http://localhost:8443
    endpoint: http://localhost:9443
    refresh: 5s
    default_view: overview
    output: tui
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProfiles(path)
	if err == nil || !strings.Contains(err.Error(), "defines both server and endpoint with different values") {
		t.Fatalf("expected conflicting server endpoint error, got %v", err)
	}
}

func TestLoadProfilesRequiresProfileFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(`profiles:
  local:
    server: http://localhost:8443
    default_view: overview
    output: tui
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProfiles(path)
	if err == nil || !strings.Contains(err.Error(), `profile "local" is missing required value: refresh`) {
		t.Fatalf("expected missing profile refresh error, got %v", err)
	}
}

func TestLoadConfigMissingDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_, err := LoadConfig("")
	if err == nil {
		t.Fatal("expected missing config error")
	}
	if !strings.Contains(err.Error(), filepath.Join(home, ".ironroot", "config")) || !strings.Contains(err.Error(), "is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	if !IsMissingDefaultConfig(err) {
		t.Fatalf("expected missing default config sentinel, got %v", err)
	}
}

func TestLoadConfigRejectsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("server: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "parse irtop config file") {
		t.Fatalf("expected invalid YAML error, got %v", err)
	}
}

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(`profiles:
  local:
    endpoint: http://localhost:8443
    refresh: 5s
    default_view: overview
    output: tui
    surprise: true
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "field surprise not found") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestLoadConfigRequiresProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(`endpoint: http://localhost:8443
refresh: 5s
default_view: overview
output: tui
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "field endpoint not found") {
		t.Fatalf("expected profile-only syntax error, got %v", err)
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
