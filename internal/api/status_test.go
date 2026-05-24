package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/config"
	"github.com/parisnakitakejser/ironroot/internal/db"
)

func TestStatusEndpointsDoNotExposeTokenSecrets(t *testing.T) {
	now := time.Now().UTC()
	store := fakeStatusStore{
		tokens: []db.BootstrapToken{{
			ID:        "token-id",
			TokenHash: "secret-hash-must-not-leak",
			Hostname:  "demo.local",
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		}},
	}
	router := NewRouter(Dependencies{Config: config.Default(), Store: store})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/status/tokens", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "secret-hash-must-not-leak") {
		t.Fatalf("token hash leaked in response: %s", rr.Body.String())
	}
	var tokens []tokenStatusItem
	if err := json.Unmarshal(rr.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].ID != "token-id" || tokens[0].Host != "demo.local" {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
}

func TestStatusOverview(t *testing.T) {
	store := fakeStatusStore{
		certs:       []db.IssuedCertificate{{Serial: "01", DNSNames: "demo.local", NotAfter: time.Now().UTC().Add(24 * time.Hour), CreatedAt: time.Now().UTC()}},
		enrollments: []db.Enrollment{{ID: "en-1", Hostname: "demo.local", CreatedAt: time.Now().UTC()}},
	}
	router := NewRouter(Dependencies{Config: config.Default(), Store: store})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/status/overview", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var overview statusOverviewResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if overview.Server.Status != "healthy" {
		t.Fatalf("unexpected server status %q", overview.Server.Status)
	}
	if overview.Certificates.Active != 1 {
		t.Fatalf("unexpected active certificate count %d", overview.Certificates.Active)
	}
}

type fakeStatusStore struct {
	tokens      []db.BootstrapToken
	certs       []db.IssuedCertificate
	enrollments []db.Enrollment
	auditLogs   []db.AuditLog
}

func (f fakeStatusStore) Close() error { return nil }
func (f fakeStatusStore) Migrate(context.Context) error {
	return nil
}
func (f fakeStatusStore) CreateBootstrapToken(context.Context, db.BootstrapToken) error {
	return nil
}
func (f fakeStatusStore) GetBootstrapTokenByHash(context.Context, string) (db.BootstrapToken, error) {
	return db.BootstrapToken{}, db.ErrNotFound
}
func (f fakeStatusStore) ListBootstrapTokens(context.Context) ([]db.BootstrapToken, error) {
	return f.tokens, nil
}
func (f fakeStatusStore) GetBootstrapToken(context.Context, string) (db.BootstrapToken, error) {
	return db.BootstrapToken{}, db.ErrNotFound
}
func (f fakeStatusStore) BootstrapTokenUsage(context.Context, string) (db.TokenUsage, error) {
	return db.TokenUsage{}, nil
}
func (f fakeStatusStore) RevokeBootstrapToken(context.Context, string) error {
	return nil
}
func (f fakeStatusStore) CreateEnrollment(context.Context, db.Enrollment) error {
	return nil
}
func (f fakeStatusStore) GetEnrollment(context.Context, string) (db.Enrollment, error) {
	return db.Enrollment{}, db.ErrNotFound
}
func (f fakeStatusStore) ListEnrollments(context.Context) ([]db.Enrollment, error) {
	return f.enrollments, nil
}
func (f fakeStatusStore) StoreIssuedCertificate(context.Context, db.IssuedCertificate) error {
	return nil
}
func (f fakeStatusStore) GetIssuedCertificate(context.Context, string) (db.IssuedCertificate, error) {
	return db.IssuedCertificate{}, db.ErrNotFound
}
func (f fakeStatusStore) ListIssuedCertificates(context.Context) ([]db.IssuedCertificate, error) {
	return f.certs, nil
}
func (f fakeStatusStore) RevokeCertificate(context.Context, db.RevokedCertificate) error {
	return nil
}
func (f fakeStatusStore) CreateAuditLog(context.Context, db.AuditLog) error {
	return nil
}
func (f fakeStatusStore) ListAuditLogs(context.Context, int) ([]db.AuditLog, error) {
	return f.auditLogs, nil
}
func (f fakeStatusStore) UpsertCAConfig(context.Context, db.CAConfig) error {
	return nil
}
func (f fakeStatusStore) ListCAConfigs(context.Context) ([]db.CAConfig, error) {
	return nil, errors.New("not configured")
}
