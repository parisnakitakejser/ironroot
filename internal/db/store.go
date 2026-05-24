package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/parisnakitakejser/ironroot/internal/config"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	Close() error
	Migrate(context.Context) error
	CreateBootstrapToken(context.Context, BootstrapToken) error
	GetBootstrapTokenByHash(context.Context, string) (BootstrapToken, error)
	ListBootstrapTokens(context.Context) ([]BootstrapToken, error)
	GetBootstrapToken(context.Context, string) (BootstrapToken, error)
	BootstrapTokenUsage(context.Context, string) (TokenUsage, error)
	RevokeBootstrapToken(context.Context, string) error
	CreateEnrollment(context.Context, Enrollment) error
	GetEnrollment(context.Context, string) (Enrollment, error)
	ListEnrollments(context.Context) ([]Enrollment, error)
	StoreIssuedCertificate(context.Context, IssuedCertificate) error
	GetIssuedCertificate(context.Context, string) (IssuedCertificate, error)
	ListIssuedCertificates(context.Context) ([]IssuedCertificate, error)
	RevokeCertificate(context.Context, RevokedCertificate) error
	CreateAuditLog(context.Context, AuditLog) error
	ListAuditLogs(context.Context, int) ([]AuditLog, error)
	UpsertCAConfig(context.Context, CAConfig) error
	ListCAConfigs(context.Context) ([]CAConfig, error)
}

type SQLStore struct{ db *sql.DB }

func Open(ctx context.Context, cfg config.DatabaseConfig) (*SQLStore, error) {
	if cfg.Driver != "sqlite" {
		return nil, fmt.Errorf("database driver %q is not implemented yet", cfg.Driver)
	}
	db, err := sql.Open("sqlite3", cfg.DSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return &SQLStore{db: db}, nil
}

func (s *SQLStore) Close() error { return s.db.Close() }

type BootstrapToken struct {
	ID        string
	TokenHash string
	Hostname  string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type TokenUsage struct {
	UsageCount     int
	LastUsedAt     *time.Time
	EnrollmentID   string
	EnrollmentHost string
}

type Enrollment struct {
	ID        string
	Hostname  string
	MachineID string
	MAC       string
	TokenID   string
	CreatedAt time.Time
}

type IssuedCertificate struct {
	Serial       string
	CAID         string
	EnrollmentID string
	Subject      string
	DNSNames     string
	PEM          string
	NotBefore    time.Time
	NotAfter     time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}

type RevokedCertificate struct {
	Serial    string
	Reason    string
	RevokedAt time.Time
}

type AuditLog struct {
	ID        string
	Action    string
	Actor     string
	Target    string
	Metadata  string
	TraceID   string
	CreatedAt time.Time
}

type CAConfig struct {
	CAID                    string
	Name                    string
	RootFingerprint         string
	IntermediateFingerprint string
	Status                  string
	CreatedAt               time.Time
	NotBefore               time.Time
	NotAfter                time.Time
}
