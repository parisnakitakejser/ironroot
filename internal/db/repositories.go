package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/telemetry"
)

func (s *SQLStore) CreateBootstrapToken(ctx context.Context, t BootstrapToken) error {
	ctx, span := telemetry.StartSpan(ctx, "db.create_bootstrap_token")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO bootstrap_tokens(id, token_hash, hostname, expires_at, revoked_at, created_at) VALUES(?,?,?,?,?,?)`,
		t.ID, t.TokenHash, t.Hostname, t.ExpiresAt, t.RevokedAt, t.CreatedAt)
	telemetry.RecordDatabase(ctx, "create_bootstrap_token", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) GetBootstrapTokenByHash(ctx context.Context, hash string) (BootstrapToken, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.get_bootstrap_token")
	started := time.Now()
	var t BootstrapToken
	err := s.db.QueryRowContext(ctx, `SELECT id, token_hash, hostname, expires_at, revoked_at, created_at FROM bootstrap_tokens WHERE token_hash = ?`, hash).
		Scan(&t.ID, &t.TokenHash, &t.Hostname, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		telemetry.RecordDatabase(ctx, "get_bootstrap_token", started, ErrNotFound)
		telemetry.EndSpan(span, ErrNotFound)
		return t, ErrNotFound
	}
	telemetry.RecordDatabase(ctx, "get_bootstrap_token", started, err)
	telemetry.EndSpan(span, err)
	return t, err
}

func (s *SQLStore) ListBootstrapTokens(ctx context.Context) ([]BootstrapToken, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_bootstrap_tokens")
	started := time.Now()
	rows, err := s.db.QueryContext(ctx, `SELECT id, token_hash, hostname, expires_at, revoked_at, created_at FROM bootstrap_tokens ORDER BY created_at DESC`)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_bootstrap_tokens", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []BootstrapToken
	for rows.Next() {
		var t BootstrapToken
		if err := rows.Scan(&t.ID, &t.TokenHash, &t.Hostname, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_bootstrap_tokens", started, err)
	telemetry.EndSpan(span, err)
	return out, err
}

func (s *SQLStore) GetBootstrapToken(ctx context.Context, id string) (BootstrapToken, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.get_bootstrap_token_by_id")
	started := time.Now()
	var t BootstrapToken
	err := s.db.QueryRowContext(ctx, `SELECT id, token_hash, hostname, expires_at, revoked_at, created_at FROM bootstrap_tokens WHERE id = ?`, id).
		Scan(&t.ID, &t.TokenHash, &t.Hostname, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		telemetry.RecordDatabase(ctx, "get_bootstrap_token_by_id", started, ErrNotFound)
		telemetry.EndSpan(span, ErrNotFound)
		return t, ErrNotFound
	}
	telemetry.RecordDatabase(ctx, "get_bootstrap_token_by_id", started, err)
	telemetry.EndSpan(span, err)
	return t, err
}

func (s *SQLStore) BootstrapTokenUsage(ctx context.Context, tokenID string) (TokenUsage, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.bootstrap_token_usage")
	started := time.Now()
	var usage TokenUsage
	var lastUsed sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), MAX(created_at) FROM enrollments WHERE token_id = ?`, tokenID).
		Scan(&usage.UsageCount, &lastUsed)
	if err != nil {
		telemetry.RecordDatabase(ctx, "bootstrap_token_usage", started, err)
		telemetry.EndSpan(span, err)
		return usage, err
	}
	if lastUsed.Valid {
		usage.LastUsedAt = &lastUsed.Time
	}
	if usage.UsageCount > 0 {
		_ = s.db.QueryRowContext(ctx, `SELECT id, hostname FROM enrollments WHERE token_id = ? ORDER BY created_at DESC LIMIT 1`, tokenID).
			Scan(&usage.EnrollmentID, &usage.EnrollmentHost)
	}
	telemetry.RecordDatabase(ctx, "bootstrap_token_usage", started, nil)
	telemetry.EndSpan(span, nil)
	return usage, nil
}

func (s *SQLStore) RevokeBootstrapToken(ctx context.Context, id string) error {
	ctx, span := telemetry.StartSpan(ctx, "db.revoke_bootstrap_token")
	started := time.Now()
	res, err := s.db.ExecContext(ctx, `UPDATE bootstrap_tokens SET revoked_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err == nil {
		if rows, rowsErr := res.RowsAffected(); rowsErr == nil && rows == 0 {
			err = ErrNotFound
		}
	}
	telemetry.RecordDatabase(ctx, "revoke_bootstrap_token", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) CreateEnrollment(ctx context.Context, e Enrollment) error {
	ctx, span := telemetry.StartSpan(ctx, "db.create_enrollment")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO enrollments(id, hostname, machine_id, mac, token_id, created_at) VALUES(?,?,?,?,?,?)`,
		e.ID, e.Hostname, e.MachineID, e.MAC, e.TokenID, e.CreatedAt)
	telemetry.RecordDatabase(ctx, "create_enrollment", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) GetEnrollment(ctx context.Context, id string) (Enrollment, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.get_enrollment")
	started := time.Now()
	var e Enrollment
	err := s.db.QueryRowContext(ctx, `SELECT id, hostname, machine_id, mac, token_id, created_at FROM enrollments WHERE id = ?`, id).
		Scan(&e.ID, &e.Hostname, &e.MachineID, &e.MAC, &e.TokenID, &e.CreatedAt)
	if err == sql.ErrNoRows {
		telemetry.RecordDatabase(ctx, "get_enrollment", started, ErrNotFound)
		telemetry.EndSpan(span, ErrNotFound)
		return e, ErrNotFound
	}
	telemetry.RecordDatabase(ctx, "get_enrollment", started, err)
	telemetry.EndSpan(span, err)
	return e, err
}

func (s *SQLStore) StoreIssuedCertificate(ctx context.Context, c IssuedCertificate) error {
	ctx, span := telemetry.StartSpan(ctx, "db.store_issued_certificate")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO issued_certificates(serial, ca_id, enrollment_id, subject, dns_names, pem, not_before, not_after, revoked_at, created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		c.Serial, c.CAID, c.EnrollmentID, c.Subject, c.DNSNames, c.PEM, c.NotBefore, c.NotAfter, c.RevokedAt, c.CreatedAt)
	telemetry.RecordDatabase(ctx, "store_issued_certificate", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) GetIssuedCertificate(ctx context.Context, serial string) (IssuedCertificate, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.get_issued_certificate")
	started := time.Now()
	var c IssuedCertificate
	err := s.db.QueryRowContext(ctx, `SELECT serial, ca_id, enrollment_id, subject, dns_names, pem, not_before, not_after, revoked_at, created_at FROM issued_certificates WHERE serial = ?`, serial).
		Scan(&c.Serial, &c.CAID, &c.EnrollmentID, &c.Subject, &c.DNSNames, &c.PEM, &c.NotBefore, &c.NotAfter, &c.RevokedAt, &c.CreatedAt)
	if err == sql.ErrNoRows {
		telemetry.RecordDatabase(ctx, "get_issued_certificate", started, ErrNotFound)
		telemetry.EndSpan(span, ErrNotFound)
		return c, ErrNotFound
	}
	telemetry.RecordDatabase(ctx, "get_issued_certificate", started, err)
	telemetry.EndSpan(span, err)
	return c, err
}

func (s *SQLStore) ListIssuedCertificates(ctx context.Context) ([]IssuedCertificate, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_issued_certificates")
	started := time.Now()
	rows, err := s.db.QueryContext(ctx, `SELECT serial, ca_id, enrollment_id, subject, dns_names, pem, not_before, not_after, revoked_at, created_at FROM issued_certificates ORDER BY created_at DESC`)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_issued_certificates", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []IssuedCertificate
	for rows.Next() {
		var c IssuedCertificate
		if err := rows.Scan(&c.Serial, &c.CAID, &c.EnrollmentID, &c.Subject, &c.DNSNames, &c.PEM, &c.NotBefore, &c.NotAfter, &c.RevokedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_issued_certificates", started, err)
	telemetry.EndSpan(span, err)
	return out, err
}

func (s *SQLStore) RevokeCertificate(ctx context.Context, r RevokedCertificate) error {
	ctx, span := telemetry.StartSpan(ctx, "db.revoke_certificate")
	started := time.Now()
	if _, err := s.db.ExecContext(ctx, `UPDATE issued_certificates SET revoked_at = ? WHERE serial = ?`, r.RevokedAt, r.Serial); err != nil {
		telemetry.RecordDatabase(ctx, "revoke_certificate", started, err)
		telemetry.EndSpan(span, err)
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO revoked_certificates(serial, reason, revoked_at) VALUES(?,?,?)`, r.Serial, r.Reason, r.RevokedAt)
	telemetry.RecordDatabase(ctx, "revoke_certificate", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) CreateAuditLog(ctx context.Context, a AuditLog) error {
	ctx, span := telemetry.StartSpan(ctx, "db.create_audit_log")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_logs(id, action, actor, target, metadata, trace_id, created_at) VALUES(?,?,?,?,?,?,?)`,
		a.ID, a.Action, a.Actor, a.Target, a.Metadata, a.TraceID, a.CreatedAt)
	telemetry.RecordDatabase(ctx, "create_audit_log", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) ListAuditLogs(ctx context.Context, limit int) ([]AuditLog, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_audit_logs")
	started := time.Now()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, action, actor, target, metadata, trace_id, created_at FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_audit_logs", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []AuditLog
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.ID, &a.Action, &a.Actor, &a.Target, &a.Metadata, &a.TraceID, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_audit_logs", started, err)
	telemetry.EndSpan(span, err)
	return out, err
}

func (s *SQLStore) UpsertCAConfig(ctx context.Context, c CAConfig) error {
	ctx, span := telemetry.StartSpan(ctx, "db.upsert_ca_config")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO ca_config(ca_id, name, root_fingerprint, intermediate_fingerprint, status, created_at, not_before, not_after)
		VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(ca_id) DO UPDATE SET name=excluded.name, root_fingerprint=excluded.root_fingerprint, intermediate_fingerprint=excluded.intermediate_fingerprint, status=excluded.status, not_before=excluded.not_before, not_after=excluded.not_after`,
		c.CAID, c.Name, c.RootFingerprint, c.IntermediateFingerprint, c.Status, c.CreatedAt, c.NotBefore, c.NotAfter)
	telemetry.RecordDatabase(ctx, "upsert_ca_config", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) ListCAConfigs(ctx context.Context) ([]CAConfig, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_ca_configs")
	started := time.Now()
	rows, err := s.db.QueryContext(ctx, `SELECT ca_id, name, root_fingerprint, intermediate_fingerprint, status, created_at, not_before, not_after FROM ca_config ORDER BY created_at DESC`)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_ca_configs", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []CAConfig
	for rows.Next() {
		var c CAConfig
		if err := rows.Scan(&c.CAID, &c.Name, &c.RootFingerprint, &c.IntermediateFingerprint, &c.Status, &c.CreatedAt, &c.NotBefore, &c.NotAfter); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_ca_configs", started, err)
	telemetry.EndSpan(span, err)
	return out, err
}
