package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	apiclient "github.com/ironroot/ironroot/pkg/client"
)

func TestDefaultCertOutputDirForOS(t *testing.T) {
	linux := defaultCertOutputDirForOS("linux", "/home/alice", "demo.local")
	if linux != filepath.Join("/home/alice", ".local", "share", "ironroot", "certs", "demo.local") {
		t.Fatalf("linux path = %s", linux)
	}
	darwin := defaultCertOutputDirForOS("darwin", "/Users/alice", "demo.local")
	if darwin != filepath.Join("/Users/alice", "Library", "Application Support", "ironroot", "certs", "demo.local") {
		t.Fatalf("darwin path = %s", darwin)
	}
}

func TestWriteCertificateOutputCreatesStructure(t *testing.T) {
	out := t.TempDir()
	resp := testCertificateResponse(t)
	key := []byte("test-private-key")
	err := writeCertificateOutput(out, key, resp, certificateOutputOptions{
		DNSNames: []string{"demo.local"}, EnrollmentID: "76777b42-c89e-415e-a5a7-3c73ad144309",
		Overwrite: true, Bundle: true, Format: "pem", KeyInfo: keyMetadata{Type: "ecdsa", Curve: "p256"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tls.key", "tls.crt", "ca-chain.crt", "fullchain.crt", "metadata.json", "fingerprints.txt", "README.txt"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
	keyInfo, err := os.Stat(filepath.Join(out, "tls.key"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("tls.key mode = %v, want 0600", keyInfo.Mode().Perm())
	}
	meta, err := os.ReadFile(filepath.Join(out, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), `"enrollment_id": "76777b42-c89e-415e-a5a7-3c73ad144309"`) {
		t.Fatalf("metadata missing enrollment id: %s", meta)
	}
}

func TestWriteCertificateOutputOverwriteProtection(t *testing.T) {
	out := t.TempDir()
	resp := testCertificateResponse(t)
	if err := os.WriteFile(filepath.Join(out, "tls.key"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := writeCertificateOutput(out, []byte("new"), resp, certificateOutputOptions{
		DNSNames: []string{"demo.local"}, EnrollmentID: "76777b42-c89e-415e-a5a7-3c73ad144309",
		Bundle: true, Format: "pem",
	})
	if err == nil || !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("expected overwrite error, got %v", err)
	}
}

func testCertificateResponse(t *testing.T) apiclient.CertificateResponse {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber:          x509Serial(t),
		Subject:               pkix.Name{CommonName: "demo.local"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return apiclient.CertificateResponse{
		Serial: "01", CertPEM: certPEM, ChainPEM: certPEM, RootPEM: certPEM,
		NotBefore: tpl.NotBefore.Format(time.RFC3339), NotAfter: tpl.NotAfter.Format(time.RFC3339),
	}
}

func x509Serial(t *testing.T) *big.Int {
	t.Helper()
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	return serial
}
