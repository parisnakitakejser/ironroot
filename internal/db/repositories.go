package db

import (
	"context"
	"database/sql"
)

func (s *SQLStore) CreateBootstrapToken(ctx context.Context, t BootstrapToken) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO bootstrap_tokens(id, token_hash, hostname, expires_at, revoked_at, created_at) VALUES(?,?,?,?,?,?)`,
		t.ID, t.TokenHash, t.Hostname, t.ExpiresAt, t.RevokedAt, t.CreatedAt)
	return err
}

func (s *SQLStore) GetBootstrapTokenByHash(ctx context.Context, hash string) (BootstrapToken, error) {
	var t BootstrapToken
	err := s.db.QueryRowContext(ctx, `SELECT id, token_hash, hostname, expires_at, revoked_at, created_at FROM bootstrap_tokens WHERE token_hash = ?`, hash).
		Scan(&t.ID, &t.TokenHash, &t.Hostname, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return t, ErrNotFound
	}
	return t, err
}

func (s *SQLStore) ListBootstrapTokens(ctx context.Context) ([]BootstrapToken, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, token_hash, hostname, expires_at, revoked_at, created_at FROM bootstrap_tokens ORDER BY created_at DESC`)
	if err != nil {
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
	return out, rows.Err()
}

func (s *SQLStore) RevokeBootstrapToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE bootstrap_tokens SET revoked_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

func (s *SQLStore) CreateEnrollment(ctx context.Context, e Enrollment) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO enrollments(id, hostname, machine_id, mac, token_id, created_at) VALUES(?,?,?,?,?,?)`,
		e.ID, e.Hostname, e.MachineID, e.MAC, e.TokenID, e.CreatedAt)
	return err
}

func (s *SQLStore) GetEnrollment(ctx context.Context, id string) (Enrollment, error) {
	var e Enrollment
	err := s.db.QueryRowContext(ctx, `SELECT id, hostname, machine_id, mac, token_id, created_at FROM enrollments WHERE id = ?`, id).
		Scan(&e.ID, &e.Hostname, &e.MachineID, &e.MAC, &e.TokenID, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return e, ErrNotFound
	}
	return e, err
}

func (s *SQLStore) StoreIssuedCertificate(ctx context.Context, c IssuedCertificate) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO issued_certificates(serial, ca_id, enrollment_id, subject, dns_names, pem, not_before, not_after, revoked_at, created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		c.Serial, c.CAID, c.EnrollmentID, c.Subject, c.DNSNames, c.PEM, c.NotBefore, c.NotAfter, c.RevokedAt, c.CreatedAt)
	return err
}

func (s *SQLStore) GetIssuedCertificate(ctx context.Context, serial string) (IssuedCertificate, error) {
	var c IssuedCertificate
	err := s.db.QueryRowContext(ctx, `SELECT serial, ca_id, enrollment_id, subject, dns_names, pem, not_before, not_after, revoked_at, created_at FROM issued_certificates WHERE serial = ?`, serial).
		Scan(&c.Serial, &c.CAID, &c.EnrollmentID, &c.Subject, &c.DNSNames, &c.PEM, &c.NotBefore, &c.NotAfter, &c.RevokedAt, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return c, ErrNotFound
	}
	return c, err
}

func (s *SQLStore) ListIssuedCertificates(ctx context.Context) ([]IssuedCertificate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT serial, ca_id, enrollment_id, subject, dns_names, pem, not_before, not_after, revoked_at, created_at FROM issued_certificates ORDER BY created_at DESC`)
	if err != nil {
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
	return out, rows.Err()
}

func (s *SQLStore) RevokeCertificate(ctx context.Context, r RevokedCertificate) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE issued_certificates SET revoked_at = ? WHERE serial = ?`, r.RevokedAt, r.Serial); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO revoked_certificates(serial, reason, revoked_at) VALUES(?,?,?)`, r.Serial, r.Reason, r.RevokedAt)
	return err
}

func (s *SQLStore) CreateAuditLog(ctx context.Context, a AuditLog) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_logs(id, action, actor, target, metadata, trace_id, created_at) VALUES(?,?,?,?,?,?,?)`,
		a.ID, a.Action, a.Actor, a.Target, a.Metadata, a.TraceID, a.CreatedAt)
	return err
}

func (s *SQLStore) ListAuditLogs(ctx context.Context, limit int) ([]AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, action, actor, target, metadata, trace_id, created_at FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
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
	return out, rows.Err()
}

func (s *SQLStore) UpsertCAConfig(ctx context.Context, c CAConfig) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO ca_config(ca_id, name, root_fingerprint, intermediate_fingerprint, status, created_at, not_before, not_after)
		VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(ca_id) DO UPDATE SET name=excluded.name, root_fingerprint=excluded.root_fingerprint, intermediate_fingerprint=excluded.intermediate_fingerprint, status=excluded.status, not_before=excluded.not_before, not_after=excluded.not_after`,
		c.CAID, c.Name, c.RootFingerprint, c.IntermediateFingerprint, c.Status, c.CreatedAt, c.NotBefore, c.NotAfter)
	return err
}

func (s *SQLStore) ListCAConfigs(ctx context.Context) ([]CAConfig, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ca_id, name, root_fingerprint, intermediate_fingerprint, status, created_at, not_before, not_after FROM ca_config ORDER BY created_at DESC`)
	if err != nil {
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
	return out, rows.Err()
}
