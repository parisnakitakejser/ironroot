package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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

func (s *SQLStore) ListEnrollments(ctx context.Context) ([]Enrollment, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_enrollments")
	started := time.Now()
	rows, err := s.db.QueryContext(ctx, `SELECT id, hostname, machine_id, mac, token_id, created_at FROM enrollments ORDER BY created_at DESC`)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_enrollments", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []Enrollment
	for rows.Next() {
		var e Enrollment
		if err := rows.Scan(&e.ID, &e.Hostname, &e.MachineID, &e.MAC, &e.TokenID, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_enrollments", started, err)
	telemetry.EndSpan(span, err)
	return out, err
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

	// Retrieve the previous entry's hash
	var lastHash string
	err := s.db.QueryRowContext(ctx, `SELECT hash FROM audit_logs ORDER BY created_at DESC, id DESC LIMIT 1`).Scan(&lastHash)
	if err != nil && err != sql.ErrNoRows {
		telemetry.RecordDatabase(ctx, "create_audit_log", started, err)
		telemetry.EndSpan(span, err)
		return err
	}

	a.PrevHash = lastHash

	// Compute current hash: SHA-256 of: prev_hash + action + actor + target + metadata + trace_id + created_at
	h := sha256.New()
	_, _ = h.Write([]byte(a.PrevHash))
	_, _ = h.Write([]byte(a.Action))
	_, _ = h.Write([]byte(a.Actor))
	_, _ = h.Write([]byte(a.Target))
	_, _ = h.Write([]byte(a.Metadata))
	_, _ = h.Write([]byte(a.TraceID))
	_, _ = h.Write([]byte(a.CreatedAt.Format(time.RFC3339)))
	a.Hash = hex.EncodeToString(h.Sum(nil))

	_, err = s.db.ExecContext(ctx, `INSERT INTO audit_logs(id, action, actor, target, metadata, trace_id, prev_hash, hash, created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		a.ID, a.Action, a.Actor, a.Target, a.Metadata, a.TraceID, a.PrevHash, a.Hash, a.CreatedAt)
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
	rows, err := s.db.QueryContext(ctx, `SELECT id, action, actor, target, metadata, trace_id, prev_hash, hash, created_at FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_audit_logs", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []AuditLog
	for rows.Next() {
		var a AuditLog
		var prevHash, hash sql.NullString
		if err := rows.Scan(&a.ID, &a.Action, &a.Actor, &a.Target, &a.Metadata, &a.TraceID, &prevHash, &hash, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.PrevHash = prevHash.String
		a.Hash = hash.String
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

func (s *SQLStore) UpsertRootCA(ctx context.Context, c RootCA) error {
	ctx, span := telemetry.StartSpan(ctx, "db.upsert_root_ca")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO root_cas(id, name, environment, fingerprint, status, trust_domain, created_at, not_before, not_after)
		VALUES(?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, environment=excluded.environment, fingerprint=excluded.fingerprint, status=excluded.status, trust_domain=excluded.trust_domain, not_before=excluded.not_before, not_after=excluded.not_after`,
		c.ID, c.Name, c.Environment, c.Fingerprint, c.Status, c.TrustDomain, c.CreatedAt, c.NotBefore, c.NotAfter)
	telemetry.RecordDatabase(ctx, "upsert_root_ca", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) ListRootCAs(ctx context.Context) ([]RootCA, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_root_cas")
	started := time.Now()
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, environment, fingerprint, status, trust_domain, created_at, not_before, not_after FROM root_cas ORDER BY environment, name`)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_root_cas", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []RootCA
	for rows.Next() {
		var c RootCA
		if err := rows.Scan(&c.ID, &c.Name, &c.Environment, &c.Fingerprint, &c.Status, &c.TrustDomain, &c.CreatedAt, &c.NotBefore, &c.NotAfter); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_root_cas", started, err)
	telemetry.EndSpan(span, err)
	return out, err
}

func (s *SQLStore) UpsertIntermediateCA(ctx context.Context, c IntermediateCA) error {
	ctx, span := telemetry.StartSpan(ctx, "db.upsert_intermediate_ca")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO intermediate_cas(id, root_id, name, environment, owner, namespace, fingerprint, status, max_ttl_seconds, allowed_dns, allowed_usages, require_approval, issuance_limit, renewal_allowed, created_at, not_before, not_after)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET root_id=excluded.root_id, name=excluded.name, environment=excluded.environment, owner=excluded.owner, namespace=excluded.namespace, fingerprint=excluded.fingerprint, status=excluded.status, max_ttl_seconds=excluded.max_ttl_seconds, allowed_dns=excluded.allowed_dns, allowed_usages=excluded.allowed_usages, require_approval=excluded.require_approval, issuance_limit=excluded.issuance_limit, renewal_allowed=excluded.renewal_allowed, not_before=excluded.not_before, not_after=excluded.not_after`,
		c.ID, c.RootID, c.Name, c.Environment, c.Owner, c.Namespace, c.Fingerprint, c.Status, int64(c.MaxTTL.Seconds()), c.AllowedDNS, c.AllowedUsages, c.RequireApproval, c.IssuanceLimit, c.RenewalAllowed, c.CreatedAt, c.NotBefore, c.NotAfter)
	telemetry.RecordDatabase(ctx, "upsert_intermediate_ca", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) ListIntermediateCAs(ctx context.Context) ([]IntermediateCA, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_intermediate_cas")
	started := time.Now()
	rows, err := s.db.QueryContext(ctx, `SELECT id, root_id, name, environment, owner, namespace, fingerprint, status, max_ttl_seconds, allowed_dns, allowed_usages, require_approval, issuance_limit, renewal_allowed, created_at, not_before, not_after FROM intermediate_cas ORDER BY environment, name`)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_intermediate_cas", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []IntermediateCA
	for rows.Next() {
		var c IntermediateCA
		var maxTTLSeconds int64
		if err := rows.Scan(&c.ID, &c.RootID, &c.Name, &c.Environment, &c.Owner, &c.Namespace, &c.Fingerprint, &c.Status, &maxTTLSeconds, &c.AllowedDNS, &c.AllowedUsages, &c.RequireApproval, &c.IssuanceLimit, &c.RenewalAllowed, &c.CreatedAt, &c.NotBefore, &c.NotAfter); err != nil {
			return nil, err
		}
		c.MaxTTL = time.Duration(maxTTLSeconds) * time.Second
		out = append(out, c)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_intermediate_cas", started, err)
	telemetry.EndSpan(span, err)
	return out, err
}

func (s *SQLStore) UpsertCARole(ctx context.Context, r CARole) error {
	ctx, span := telemetry.StartSpan(ctx, "db.upsert_ca_role")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO ca_roles(id, name, subject, intermediate_id, permissions, created_at)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, subject=excluded.subject, intermediate_id=excluded.intermediate_id, permissions=excluded.permissions`,
		r.ID, r.Name, r.Subject, r.IntermediateID, r.Permissions, r.CreatedAt)
	telemetry.RecordDatabase(ctx, "upsert_ca_role", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) ListCARoles(ctx context.Context) ([]CARole, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_ca_roles")
	started := time.Now()
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, subject, intermediate_id, permissions, created_at FROM ca_roles ORDER BY name`)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_ca_roles", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []CARole
	for rows.Next() {
		var r CARole
		if err := rows.Scan(&r.ID, &r.Name, &r.Subject, &r.IntermediateID, &r.Permissions, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_ca_roles", started, err)
	telemetry.EndSpan(span, err)
	return out, err
}

func (s *SQLStore) UpsertCATokenPolicy(ctx context.Context, p CATokenPolicy) error {
	ctx, span := telemetry.StartSpan(ctx, "db.upsert_ca_token_policy")
	started := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO ca_token_policies(id, name, intermediate_id, certificate_types, allowed_dns, max_ttl_seconds, issuance_limit, renewal_allowed, require_approval, created_at, expires_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, intermediate_id=excluded.intermediate_id, certificate_types=excluded.certificate_types, allowed_dns=excluded.allowed_dns, max_ttl_seconds=excluded.max_ttl_seconds, issuance_limit=excluded.issuance_limit, renewal_allowed=excluded.renewal_allowed, require_approval=excluded.require_approval, expires_at=excluded.expires_at`,
		p.ID, p.Name, p.IntermediateID, p.CertificateTypes, p.AllowedDNS, int64(p.MaxTTL.Seconds()), p.IssuanceLimit, p.RenewalAllowed, p.RequireApproval, p.CreatedAt, p.ExpiresAt)
	telemetry.RecordDatabase(ctx, "upsert_ca_token_policy", started, err)
	telemetry.EndSpan(span, err)
	return err
}

func (s *SQLStore) ListCATokenPolicies(ctx context.Context) ([]CATokenPolicy, error) {
	ctx, span := telemetry.StartSpan(ctx, "db.list_ca_token_policies")
	started := time.Now()
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, intermediate_id, certificate_types, allowed_dns, max_ttl_seconds, issuance_limit, renewal_allowed, require_approval, created_at, expires_at FROM ca_token_policies ORDER BY name`)
	if err != nil {
		telemetry.RecordDatabase(ctx, "list_ca_token_policies", started, err)
		telemetry.EndSpan(span, err)
		return nil, err
	}
	defer rows.Close()
	var out []CATokenPolicy
	for rows.Next() {
		var p CATokenPolicy
		var maxTTLSeconds int64
		if err := rows.Scan(&p.ID, &p.Name, &p.IntermediateID, &p.CertificateTypes, &p.AllowedDNS, &maxTTLSeconds, &p.IssuanceLimit, &p.RenewalAllowed, &p.RequireApproval, &p.CreatedAt, &p.ExpiresAt); err != nil {
			return nil, err
		}
		p.MaxTTL = time.Duration(maxTTLSeconds) * time.Second
		out = append(out, p)
	}
	err = rows.Err()
	telemetry.RecordDatabase(ctx, "list_ca_token_policies", started, err)
	telemetry.EndSpan(span, err)
	return out, err
}
