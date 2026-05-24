package irtop

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSnapshotParsesStatusEndpointsAndSendsToken(t *testing.T) {
	var authHeader string
	client := &Client{
		baseURL: "https://ironroot.test",
		token:   "read-only-token",
		http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body := responseBodyForPath(r.URL.Path)
			if body == "" {
				return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
			}
			authHeader = r.Header.Get("Authorization")
			return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})},
	}
	snapshot, err := client.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if authHeader != "Bearer read-only-token" {
		t.Fatalf("unexpected auth header %q", authHeader)
	}
	if snapshot.Overview.Server.Status != "healthy" {
		t.Fatalf("unexpected overview status %q", snapshot.Overview.Server.Status)
	}
	if len(snapshot.Certificates) != 1 || snapshot.Certificates[0].Serial != "01" {
		t.Fatalf("unexpected certificates %#v", snapshot.Certificates)
	}
	if len(snapshot.Tokens) != 1 || snapshot.Tokens[0].ID != "tok-1" {
		t.Fatalf("unexpected tokens %#v", snapshot.Tokens)
	}
}

func responseBodyForPath(path string) string {
	mustJSON := func(v any) string {
		b, _ := json.Marshal(v)
		return string(b)
	}
	switch path {
	case "/v1/status/overview":
		return mustJSON(Overview{
			Server:       ServerStatus{Status: "healthy", APIHealth: "ok", Readiness: "ready"},
			Certificates: CertificateSummary{Active: 2},
			Telemetry:    Telemetry{Enabled: true, ExporterStatus: "ready"},
		})
	case "/v1/status/server":
		return mustJSON(ServerStatus{Status: "healthy", Version: "v0.1.0"})
	case "/v1/status/ca":
		return mustJSON(CAStatus{ChainStatus: "valid"})
	case "/v1/status/certificates":
		return mustJSON([]Certificate{{Serial: "01", DNSNames: []string{"demo.local"}, Status: "active"}})
	case "/v1/status/enrollments":
		return mustJSON([]Enrollment{{ID: "en-1", Hostname: "demo.local"}})
	case "/v1/status/tokens":
		return mustJSON([]Token{{ID: "tok-1", Host: "demo.local", Status: "active"}})
	case "/v1/status/security":
		return mustJSON(Security{Status: "pass"})
	case "/v1/status/telemetry":
		return mustJSON(Telemetry{Enabled: true})
	case "/v1/audit/recent":
		return mustJSON([]AuditEvent{{Action: "certificate.issued", Result: "success"}})
	default:
		return ""
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestMaskToken(t *testing.T) {
	if got := MaskToken("short"); got != "********" {
		t.Fatalf("short token mask = %q", got)
	}
	if got := MaskToken("abcdefghijkl"); got != "abcd...ijkl" {
		t.Fatalf("long token mask = %q", got)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []Config{
		{Server: ""},
		{Server: "localhost:8443"},
		{Server: "ftp://localhost:8443"},
		{Server: "https://"},
	}
	for _, cfg := range tests {
		if err := ValidateConfig(cfg); err == nil {
			t.Fatalf("expected validation error for %#v", cfg)
		}
	}
	for _, cfg := range []Config{{Server: "http://localhost:8443"}, {Server: "https://localhost:8443"}} {
		if err := ValidateConfig(cfg); err != nil {
			t.Fatalf("unexpected validation error for %#v: %v", cfg, err)
		}
	}
}

func TestFriendlyConnectionErrorMapsHTTPSHTTPMismatch(t *testing.T) {
	err := FriendlyConnectionError("https://localhost:8443", errString("http: server gave HTTP response to HTTPS client"))
	if err == nil || !strings.Contains(err.Error(), "try:\nirtop --server http://localhost:8443") {
		t.Fatalf("unexpected friendly error: %v", err)
	}
}

func TestTLSConfigLoadsCAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(path, testCA(t), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := TLSConfig(Config{Server: "https://localhost:8443", CAFile: path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || cfg.RootCAs == nil {
		t.Fatal("expected root CA pool")
	}
}

func TestTLSConfigRejectsInvalidCAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := TLSConfig(Config{Server: "https://localhost:8443", CAFile: path}); err == nil {
		t.Fatal("expected invalid CA file error")
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func testCA(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "IronRoot Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
