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
