package ca

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"path/filepath"
	"testing"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/config"
	"github.com/parisnakitakejser/ironroot/internal/testutil"
)

func TestLoadAuthorityAndSignCSR(t *testing.T) {
	fixture := testutil.NewTestCA(t)
	auth, err := LoadAuthority(config.PKIConfig{
		RootFile: filepath.Join(fixture.Dir, "root-ca.crt"), ChainFile: filepath.Join(fixture.Dir, "ca-chain.crt"), IntermediateCertFile: filepath.Join(fixture.Dir, "intermediate.crt"), IntermediateKeyFile: filepath.Join(fixture.Dir, "intermediate.key"),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, csr, err := GenerateKeyAndCSR("node", []string{"node.local"})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := auth.SignCSR(context.Background(), string(csr), []string{"node.local"}, 90*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode([]byte(issued.CertPEM))
	if block == nil {
		t.Fatal("missing cert pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if cert.DNSNames[0] != "node.local" {
		t.Fatalf("dns = %v", cert.DNSNames)
	}
}
