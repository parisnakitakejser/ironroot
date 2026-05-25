package db

import (
	"context"
	"testing"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/config"
)

func TestMigrateAndRepositories(t *testing.T) {
	store, err := Open(context.Background(), config.DatabaseConfig{Driver: "sqlite", DSN: "file:" + t.TempDir() + "/test.db?_foreign_keys=on"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	token := BootstrapToken{ID: "tok-1", TokenHash: "hash", Hostname: "node-01", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}
	if err := store.CreateBootstrapToken(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetBootstrapTokenByHash(context.Background(), "hash")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hostname != "node-01" {
		t.Fatalf("hostname = %s", got.Hostname)
	}
}

func TestMultiCARepositories(t *testing.T) {
	store, err := Open(context.Background(), config.DatabaseConfig{Driver: "sqlite", DSN: "file:" + t.TempDir() + "/test.db?_foreign_keys=on"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	root := RootCA{ID: "root-prod", Name: "Production Root", Environment: "production", Fingerprint: "root-fp", Status: "active", TrustDomain: "prod.example.com", CreatedAt: now, NotBefore: now, NotAfter: now.Add(10 * 365 * 24 * time.Hour)}
	if err := store.UpsertRootCA(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	intermediate := IntermediateCA{ID: "int-web", RootID: root.ID, Name: "Web Intermediate", Environment: "production", Owner: "platform", Namespace: "web", Fingerprint: "int-fp", Status: "active", MaxTTL: 90 * 24 * time.Hour, AllowedDNS: "*.prod.example.com", AllowedUsages: "server,client", RenewalAllowed: true, CreatedAt: now, NotBefore: now, NotAfter: now.Add(365 * 24 * time.Hour)}
	if err := store.UpsertIntermediateCA(context.Background(), intermediate); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCARole(context.Background(), CARole{ID: "role-web", Name: "web-issuer", Subject: "team:platform", IntermediateID: intermediate.ID, Permissions: "request,renew", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCATokenPolicy(context.Background(), CATokenPolicy{ID: "policy-web", Name: "web-short", IntermediateID: intermediate.ID, CertificateTypes: "server", AllowedDNS: "*.prod.example.com", MaxTTL: 24 * time.Hour, IssuanceLimit: 5, RenewalAllowed: true, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	roots, err := store.ListRootCAs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || roots[0].Environment != "production" {
		t.Fatalf("unexpected roots: %#v", roots)
	}
	intermediates, err := store.ListIntermediateCAs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(intermediates) != 1 || intermediates[0].MaxTTL != 90*24*time.Hour {
		t.Fatalf("unexpected intermediates: %#v", intermediates)
	}
	roles, err := store.ListCARoles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 || roles[0].Permissions != "request,renew" {
		t.Fatalf("unexpected roles: %#v", roles)
	}
	policies, err := store.ListCATokenPolicies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(policies) != 1 || policies[0].IssuanceLimit != 5 {
		t.Fatalf("unexpected policies: %#v", policies)
	}
}
