package dev

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveBaseDirDefaultsToCurrentDirectory(t *testing.T) {
	dir := realTempDir(t)
	oldwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	got, err := resolveBaseDir("")
	if err != nil {
		t.Fatalf("resolveBaseDir failed: %v", err)
	}
	if got != dir {
		t.Fatalf("base dir = %q, want %q", got, dir)
	}
}

func TestDevInitWorksOutsideCheckout(t *testing.T) {
	base := realTempDir(t)
	var out bytes.Buffer

	if err := RunDevInit(t.Context(), DevInitOptions{BaseDir: base, Output: ".localdev", Stdout: &out}); err != nil {
		t.Fatalf("RunDevInit failed outside checkout: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".localdev", "config", "config.yaml")); err != nil {
		t.Fatalf("missing generated config: %v", err)
	}
	if !strings.Contains(out.String(), "IronRoot local development workspace initialized") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestDevInitCreatesWorkspaceAndGeneratedConfig(t *testing.T) {
	base := realTempDir(t)
	var out bytes.Buffer

	err := RunDevInit(t.Context(), DevInitOptions{BaseDir: base, Output: ".localdev", Stdout: &out})
	if err != nil {
		t.Fatalf("RunDevInit failed: %v", err)
	}

	workspace := filepath.Join(base, ".localdev")
	for _, rel := range []string{"config", "data", "pki", "pki/root", "pki/intermediate", "certs", "logs", "tmp"} {
		if info, err := os.Stat(filepath.Join(workspace, rel)); err != nil || !info.IsDir() {
			t.Fatalf("missing directory %s: %v", rel, err)
		}
	}
	configPath := filepath.Join(workspace, "config", "config.yaml")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), filepath.ToSlash(filepath.Join(workspace, "data", "ironroot.db"))) {
		t.Fatalf("config does not contain generated absolute database path:\n%s", config)
	}
	if !strings.Contains(string(config), filepath.ToSlash(filepath.Join(workspace, "pki", "root", "root-ca.crt"))) {
		t.Fatalf("config does not contain generated absolute root path:\n%s", config)
	}
	if !strings.Contains(out.String(), "Base directory:") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestDevInitIsIdempotentAndDoesNotOverwriteWithoutForce(t *testing.T) {
	base := realTempDir(t)
	if err := RunDevInit(t.Context(), DevInitOptions{BaseDir: base, Output: ".localdev", Stdout: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(base, ".localdev", "config", "config.yaml")
	if err := os.WriteFile(configPath, []byte("custom: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := RunDevInit(t.Context(), DevInitOptions{BaseDir: base, Output: ".localdev", Stdout: &out}); err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(config) != "custom: true\n" {
		t.Fatalf("config was overwritten without --force: %s", config)
	}
	if !strings.Contains(out.String(), "Skipped existing files") {
		t.Fatalf("expected skipped files output, got: %s", out.String())
	}

	if err := RunDevInit(t.Context(), DevInitOptions{BaseDir: base, Output: ".localdev", Force: true, Stdout: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	config, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(config) == "custom: true\n" {
		t.Fatal("config was not regenerated with --force")
	}
}

func TestDevInitDryRunDoesNotWrite(t *testing.T) {
	base := realTempDir(t)
	var out bytes.Buffer

	if err := RunDevInit(t.Context(), DevInitOptions{BaseDir: base, Output: ".localdev", DryRun: true, Stdout: &out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, ".localdev")); !os.IsNotExist(err) {
		t.Fatalf("dry run created workspace, stat err: %v", err)
	}
	if !strings.Contains(out.String(), "DRY RUN") {
		t.Fatalf("expected dry-run output, got: %s", out.String())
	}
}

func TestDevInitExplicitBaseDirectory(t *testing.T) {
	base := realTempDir(t)
	oldwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(realTempDir(t)); err != nil {
		t.Fatal(err)
	}

	if err := RunDevInit(t.Context(), DevInitOptions{BaseDir: base, Output: ".localdev", Stdout: &bytes.Buffer{}}); err != nil {
		t.Fatalf("RunDevInit with --base-dir failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".localdev", "config", "config.yaml")); err != nil {
		t.Fatalf("missing generated config: %v", err)
	}
}

func realTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	realDir, err := filepath.EvalSymlinks(dir)
	if err == nil {
		return realDir
	}
	return dir
}
