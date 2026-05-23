package ca

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ironroot/ironroot/internal/config"
)

func TestCreateRootIntermediateAndVerifyLeaf(t *testing.T) {
	dir := t.TempDir()
	rootDir := filepath.Join(dir, "root")
	intermediateDir := filepath.Join(dir, "intermediate")

	root, err := CreateRoot(CreateRootOptions{
		Name: "Test Root", OutDir: rootDir, Password: "root-pass", Lifetime: 20 * 365 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	intermediate, err := CreateIntermediate(CreateIntermediateOptions{
		Name: "Test Intermediate", OutDir: intermediateDir, RootCertPath: root.CertPath, RootKeyPath: root.KeyPath,
		RootPassword: "root-pass", Password: "intermediate-pass", Lifetime: 5 * 365 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyChain(root.CertPath, intermediate.CertPath, ""); err != nil {
		t.Fatal(err)
	}

	auth, err := LoadAuthority(configForTest(root.CertPath, intermediate))
	if err != nil {
		t.Fatal(err)
	}
	_, csr, err := GenerateKeyAndCSR("demo.home.arpa", []string{"demo.home.arpa"})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := auth.SignCSR(context.Background(), string(csr), []string{"demo.home.arpa"}, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	leafPath := filepath.Join(dir, "tls.crt")
	if err := os.WriteFile(leafPath, []byte(issued.CertPEM), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChain(root.CertPath, intermediate.CertPath, leafPath); err != nil {
		t.Fatal(err)
	}
}

func configForTest(rootPath string, intermediate CreateIntermediateResult) config.PKIConfig {
	return config.PKIConfig{
		RootFile: rootPath, ChainFile: intermediate.ChainPath, IntermediateCertFile: intermediate.CertPath,
		IntermediateKeyFile: intermediate.KeyPath, IntermediateKeyPass: "intermediate-pass",
	}
}
