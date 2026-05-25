package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/irtop"
)

func TestLoadProfilesForFlagsAllowsServerWithoutDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	profiles, err := loadProfilesForFlags("", "", "http://localhost:8443")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles.Profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(profiles.Profiles))
	}
	if profiles.ActiveConfig().Server != "http://localhost:8443" {
		t.Fatalf("server = %q", profiles.ActiveConfig().Server)
	}
}

func TestLoadProfilesForFlagsRequiresExplicitConfigPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, err := loadProfilesForFlags(missing, "", "http://localhost:8443")
	if err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("expected explicit missing config error, got %v", err)
	}
}

func TestLoadProfilesForFlagsRequiresConfigForProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	_, err := loadProfilesForFlags("", "production", "http://localhost:8443")
	if err == nil || !strings.Contains(err.Error(), ".ironroot") {
		t.Fatalf("expected missing default config error, got %v", err)
	}
}

func TestApplyFlagOverrides(t *testing.T) {
	cfg := applyFlagOverrides(irtop.Config{
		Server:      "https://configured.example",
		Refresh:     time.Second,
		DefaultView: "overview",
		Output:      "",
	}, "http://localhost:8443", "/tmp/root-ca.crt", 2*time.Second, true, "read-token", "text")

	if cfg.Server != "http://localhost:8443" {
		t.Fatalf("server = %q", cfg.Server)
	}
	if cfg.CAFile != "/tmp/root-ca.crt" {
		t.Fatalf("ca file = %q", cfg.CAFile)
	}
	if cfg.Refresh != 2*time.Second {
		t.Fatalf("refresh = %s", cfg.Refresh)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected insecure skip verify override")
	}
	if cfg.Token != "read-token" {
		t.Fatalf("token = %q", cfg.Token)
	}
	if cfg.Output != "text" {
		t.Fatalf("output = %q", cfg.Output)
	}
}

func TestApplyFlagOverridesDefaultsOutput(t *testing.T) {
	cfg := applyFlagOverrides(irtop.Config{Server: "http://localhost:8443"}, "", "", 0, false, "", "")
	if cfg.Output != "tui" {
		t.Fatalf("output = %q, want tui", cfg.Output)
	}
}
