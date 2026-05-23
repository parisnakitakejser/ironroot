package e2e

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ironroot/ironroot/internal/api"
	"github.com/ironroot/ironroot/internal/audit"
	"github.com/ironroot/ironroot/internal/ca"
	"github.com/ironroot/ironroot/internal/config"
	ironcrypto "github.com/ironroot/ironroot/internal/crypto"
	"github.com/ironroot/ironroot/internal/db"
	"github.com/ironroot/ironroot/internal/telemetry"
	"github.com/ironroot/ironroot/internal/testutil"
	apiclient "github.com/ironroot/ironroot/pkg/client"
)

func TestCertificateLifecycle(t *testing.T) {
	_, err := telemetry.Configure(context.Background(), config.TelemetryConfig{Enabled: false}, "e2e")
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(context.Background(), config.DatabaseConfig{Driver: "sqlite", DSN: "file:" + t.TempDir() + "/e2e.db?_foreign_keys=on"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	fixture := testutil.NewTestCA(t)
	auth, err := ca.LoadAuthority(config.PKIConfig{
		RootFile: filepath.Join(fixture.Dir, "root-ca.crt"), ChainFile: filepath.Join(fixture.Dir, "ca-chain.crt"), IntermediateCertFile: filepath.Join(fixture.Dir, "intermediate.crt"), IntermediateKeyFile: filepath.Join(fixture.Dir, "intermediate.key"),
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.PKI.DefaultLifetime = 90 * 24 * time.Hour
	cfg.PKI.RenewBefore = 30 * 24 * time.Hour
	server := httptest.NewServer(api.NewRouter(api.Dependencies{Config: cfg, Store: store, Authority: auth, Audit: audit.New(store), Logger: telemetry.NewLogger(testingWriter{t}, "debug")}))
	defer server.Close()
	client := apiclient.New(server.URL)

	token, _ := ironcrypto.NewBootstrapToken()
	if err := store.CreateBootstrapToken(context.Background(), db.BootstrapToken{ID: uuid.NewString(), TokenHash: ironcrypto.HashToken(token), Hostname: "node", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	enroll, err := client.Enroll(context.Background(), apiclient.EnrollmentRequest{Token: token, Hostname: "node", MachineID: "machine"})
	if err != nil {
		t.Fatal(err)
	}
	_, csr, err := ca.GenerateKeyAndCSR("node", []string{"node.local"})
	if err != nil {
		t.Fatal(err)
	}
	cert, err := client.RequestCertificate(context.Background(), apiclient.CertificateRequest{EnrollmentID: enroll.EnrollmentID, CSRPEM: string(csr), DNSNames: []string{"node.local"}})
	if err != nil {
		t.Fatal(err)
	}
	if cert.Serial == "" {
		t.Fatal("missing serial")
	}
	block, _ := pem.Decode([]byte(cert.CertPEM))
	if block == nil {
		t.Fatal("missing cert pem")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RenewCertificate(context.Background(), apiclient.CertificateRequest{EnrollmentID: enroll.EnrollmentID, CSRPEM: string(csr), DNSNames: []string{"node.local"}}); err != nil {
		t.Fatal(err)
	}
	if err := client.RevokeCertificate(context.Background(), cert.Serial, "e2e"); err != nil {
		t.Fatal(err)
	}
	logs, err := client.Audit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) == 0 {
		t.Fatal("missing audit logs")
	}
}

type testingWriter struct{ t *testing.T }

func (w testingWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}
