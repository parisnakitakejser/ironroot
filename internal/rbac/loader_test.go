package rbac

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/config"
	"github.com/parisnakitakejser/ironroot/internal/db"
)

func TestLoadAndApplyMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	rbacDir := filepath.Join(dir, "config", "rbac")
	if err := os.MkdirAll(rbacDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(rbacDir, "20-binding.yaml"), `apiVersion: ironroot.io/v1alpha1
kind: CARoleBinding
metadata:
  name: platform-web-binding
spec:
  roleRef:
    kind: CARole
    name: platform-web-issuer
  subjects:
    - kind: Group
      name: platform
---
apiVersion: ironroot.io/v1alpha1
kind: TokenPolicy
metadata:
  name: local-short-lived
spec:
  intermediateRef: int-local
  certificateTypes:
    - server
  allowedDNS:
    - demo.local
  maxTTL: 1h
  issuanceLimit: 5
  renewalAllowed: true
  requireApproval: false
  expiresAt: 2030-01-01T00:00:00Z
`)
	writeFile(t, filepath.Join(rbacDir, "10-role.yaml"), `apiVersion: ironroot.io/v1alpha1
kind: Group
metadata:
  name: platform
---
apiVersion: ironroot.io/v1alpha1
kind: CARole
metadata:
  name: platform-web-issuer
spec:
  rules:
    - resources:
        - certificates
      verbs:
        - request
        - renew
      intermediateRef: int-local
`)

	store := newTestStore(t)
	result, err := LoadAndApply(context.Background(), config.RBACConfig{
		Enabled: true,
		Mode:    "file",
		Paths:   []string{filepath.Join(rbacDir, "*.yaml")},
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 2 || filepath.Base(result.Files[0]) != "10-role.yaml" || filepath.Base(result.Files[1]) != "20-binding.yaml" {
		t.Fatalf("files not loaded deterministically: %#v", result.Files)
	}
	if result.Groups != 1 || result.CARoles != 1 || result.CARoleBindings != 1 || result.TokenPolicies != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	roles, err := store.ListCARoles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 || roles[0].Subject != "group:platform" || roles[0].IntermediateID != "int-local" || roles[0].Permissions != "request,renew" {
		t.Fatalf("unexpected roles: %#v", roles)
	}
	policies, err := store.ListCATokenPolicies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(policies) != 1 || policies[0].MaxTTL != time.Hour || policies[0].AllowedDNS != "demo.local" {
		t.Fatalf("unexpected policies: %#v", policies)
	}
}

func TestLoadAndApplyRejectsInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	writeFile(t, path, `apiVersion: ironroot.io/v1alpha1
kind: TokenPolicy
metadata:
  name: missing-intermediate
spec:
  certificateTypes:
    - server
`)

	_, err := LoadAndApply(context.Background(), config.RBACConfig{Enabled: true, Mode: "file", Paths: []string{path}}, newTestStore(t))
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func newTestStore(t *testing.T) *db.SQLStore {
	t.Helper()
	store, err := db.Open(context.Background(), config.DatabaseConfig{Driver: "sqlite", DSN: "file:" + filepath.Join(t.TempDir(), "test.db") + "?_foreign_keys=on"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.UpsertRootCA(context.Background(), db.RootCA{ID: "root-local", Name: "Local Root", Environment: "development", Fingerprint: "root", Status: "active", CreatedAt: now, NotBefore: now, NotAfter: now.Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertIntermediateCA(context.Background(), db.IntermediateCA{ID: "int-local", RootID: "root-local", Name: "Local Intermediate", Environment: "development", Fingerprint: "intermediate", Status: "active", CreatedAt: now, NotBefore: now, NotAfter: now.Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	return store
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
