package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type TestCA struct {
	Dir             string
	RootCertPEM     []byte
	IntermediatePEM []byte
	IntermediateKey []byte
}

func NewTestCA(t *testing.T) TestCA {
	t.Helper()
	dir := t.TempDir()
	rootKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rootTpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "IronRoot Test Root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(20 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	rootDER, _ := x509.CreateCertificate(rand.Reader, rootTpl, rootTpl, &rootKey.PublicKey, rootKey)
	rootPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER})
	rootCert, _ := x509.ParseCertificate(rootDER)

	intKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	intTpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "IronRoot Test Intermediate"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(5 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	intDER, _ := x509.CreateCertificate(rand.Reader, intTpl, rootCert, &intKey.PublicKey, rootKey)
	intPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: intDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(intKey)})
	write := func(name string, b []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("root-ca.crt", rootPEM)
	write("ca-chain.crt", append(intPEM, rootPEM...))
	write("intermediate.crt", intPEM)
	write("intermediate.key", keyPEM)
	return TestCA{Dir: dir, RootCertPEM: rootPEM, IntermediatePEM: intPEM, IntermediateKey: keyPEM}
}
