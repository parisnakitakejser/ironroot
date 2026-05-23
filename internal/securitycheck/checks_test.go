package securitycheck

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ironroot/ironroot/internal/config"
	"github.com/ironroot/ironroot/internal/testutil"
)

func TestFilePermissionCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permissions only")
	}
	dir := t.TempDir()
	key := filepath.Join(dir, "intermediate.key")
	if err := os.WriteFile(key, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.PKI.IntermediateKeyFile = key
	result := (FilePermissionCheck{IDValue: "x", Title: "x", CategoryValue: "ca", PathKind: "ca_key"}).Run(context.Background(), Target{Config: cfg})
	if result.Status != StatusFail {
		t.Fatalf("status = %s", result.Status)
	}
	if err := os.Chmod(key, 0o600); err != nil {
		t.Fatal(err)
	}
	result = (FilePermissionCheck{IDValue: "x", Title: "x", CategoryValue: "ca", PathKind: "ca_key"}).Run(context.Background(), Target{Config: cfg})
	if result.Status != StatusPass {
		t.Fatalf("status = %s", result.Status)
	}
}

func TestCAChainCheck(t *testing.T) {
	fixture := testutil.NewTestCA(t)
	cfg := config.Default()
	cfg.PKI.RootFile = filepath.Join(fixture.Dir, "root-ca.crt")
	cfg.PKI.IntermediateCertFile = filepath.Join(fixture.Dir, "intermediate.crt")
	result := CAChainCheck{}.Run(context.Background(), Target{Config: cfg, Now: time.Now().UTC()})
	if result.Status != StatusPass {
		t.Fatalf("status = %s message=%s", result.Status, result.Message)
	}
}

func TestEvaluateTokenTTL(t *testing.T) {
	if got := EvaluateTokenTTL(2*time.Hour, time.Hour); got.Status != StatusWarn {
		t.Fatalf("expected warn, got %s", got.Status)
	}
	if got := EvaluateTokenTTL(30*time.Minute, time.Hour); got.Status != StatusPass {
		t.Fatalf("expected pass, got %s", got.Status)
	}
}

func TestRuntimeDetectionFunctions(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("POD_NAME", "")
	if InKubernetes() {
		t.Fatal("expected kubernetes detection to be false without env")
	}
	t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	if !InKubernetes() {
		t.Fatal("expected kubernetes detection from env")
	}
	_ = InContainer()
}
