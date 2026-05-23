package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(t.TempDir() + "/missing.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("driver = %s", cfg.Database.Driver)
	}
	if cfg.PKI.DefaultLifetime == 0 || cfg.PKI.RenewBefore == 0 {
		t.Fatal("expected certificate lifetime defaults")
	}
}
