package main

import (
	"path/filepath"
	"strings"
	"testing"
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
