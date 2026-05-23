package ca

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ironroot/ironroot/internal/config"
)

func TestCreateRootAdvancedDefaults(t *testing.T) {
	dir := t.TempDir()
	root, err := CreateRoot(CreateRootOptions{
		Name: "Test Root", OutDir: dir, Password: "root-pass", EncryptKey: true, Lifetime: 20 * 365 * 24 * time.Hour,
		IsCA: true, AllowCertSigning: true, AllowCRLSigning: true,
		GenerateTrustBundle: true, WritePEM: true, WriteDER: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		root.CertPath,
		root.KeyPath,
		root.PublicKeyPath,
		root.PEMPath,
		root.DERPath,
		root.MetadataPath,
		root.FingerprintsPath,
		root.RecoveryPath,
		filepath.Join(root.TrustBundleDir, "root-ca.crt"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected output %s: %v", path, err)
		}
	}
	keyPEM, err := os.ReadFile(root.KeyPath)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil || !x509.IsEncryptedPEMBlock(block) {
		t.Fatalf("expected encrypted root private key, got block %#v", block)
	}
	info, err := InspectCertificates([]string{root.CertPath})
	if err != nil {
		t.Fatal(err)
	}
	if got := info[0].Algorithm; got != "ecdsa" {
		t.Fatalf("algorithm = %q, want ecdsa", got)
	}
	if got := info[0].Curve; got != "p384" {
		t.Fatalf("curve = %q, want p384", got)
	}
	if got := info[0].MaxPathLength; got != 1 {
		t.Fatalf("max path length = %d, want 1", got)
	}
}

func TestCreateRootRSAOptions(t *testing.T) {
	dir := t.TempDir()
	root, err := CreateRoot(CreateRootOptions{
		Name: "RSA Root", OutDir: dir, Password: "root-pass", EncryptKey: true,
		Algorithm: "rsa", RSABits: 2048, Lifetime: 365 * 24 * time.Hour,
		IsCA: true, AllowCertSigning: true, AllowCRLSigning: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	info, err := InspectCertificates([]string{root.CertPath})
	if err != nil {
		t.Fatal(err)
	}
	if got := info[0].Algorithm; got != "rsa" {
		t.Fatalf("algorithm = %q, want rsa", got)
	}
	if got := info[0].KeySize; got != 2048 {
		t.Fatalf("key size = %d, want 2048", got)
	}
}

func TestCreateRootIntermediateAndVerifyLeaf(t *testing.T) {
	dir := t.TempDir()
	rootDir := filepath.Join(dir, "root")
	intermediateDir := filepath.Join(dir, "intermediate")

	root, err := CreateRoot(CreateRootOptions{
		Name: "Test Root", OutDir: rootDir, Password: "root-pass", Lifetime: 20 * 365 * 24 * time.Hour,
		EncryptKey: true, IsCA: true, AllowCertSigning: true, AllowCRLSigning: true,
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
	info, err := InspectCertificates([]string{intermediate.CertPath})
	if err != nil {
		t.Fatal(err)
	}
	if got := info[0].Algorithm; got != "ecdsa" {
		t.Fatalf("intermediate algorithm = %q, want ecdsa", got)
	}
	if got := info[0].Curve; got != "p256" {
		t.Fatalf("intermediate curve = %q, want p256", got)
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
