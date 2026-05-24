package enrollment

import (
	"context"
	"testing"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/config"
	ironcrypto "github.com/parisnakitakejser/ironroot/internal/crypto"
	"github.com/parisnakitakejser/ironroot/internal/db"
)

func TestValidateAndEnroll(t *testing.T) {
	store, err := db.Open(context.Background(), config.DatabaseConfig{Driver: "sqlite", DSN: "file:" + t.TempDir() + "/test.db?_foreign_keys=on"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	token := "secret"
	if err := store.CreateBootstrapToken(context.Background(), db.BootstrapToken{ID: "tok", TokenHash: ironcrypto.HashToken(token), Hostname: "node", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	resp, err := (Service{Store: store}).ValidateAndEnroll(context.Background(), Request{Token: token, Hostname: "node", MachineID: "machine"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.EnrollmentID == "" {
		t.Fatal("missing enrollment id")
	}
	if _, err := (Service{Store: store}).ValidateAndEnroll(context.Background(), Request{Token: token, Hostname: "other", MachineID: "machine"}); err != ErrInvalidToken {
		t.Fatalf("expected invalid token, got %v", err)
	}
}
