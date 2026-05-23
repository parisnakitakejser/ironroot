package db

import "context"

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS ca_config (
		ca_id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		root_fingerprint TEXT NOT NULL,
		intermediate_fingerprint TEXT NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('active','disabled','retired')),
		created_at TIMESTAMP NOT NULL,
		not_before TIMESTAMP NOT NULL,
		not_after TIMESTAMP NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS bootstrap_tokens (
		id TEXT PRIMARY KEY,
		token_hash TEXT NOT NULL UNIQUE,
		hostname TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		revoked_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS enrollments (
		id TEXT PRIMARY KEY,
		hostname TEXT NOT NULL,
		machine_id TEXT NOT NULL,
		mac TEXT,
		token_id TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(token_id) REFERENCES bootstrap_tokens(id)
	);`,
	`CREATE TABLE IF NOT EXISTS issued_certificates (
		serial TEXT PRIMARY KEY,
		ca_id TEXT NOT NULL,
		enrollment_id TEXT,
		subject TEXT NOT NULL,
		dns_names TEXT,
		pem TEXT NOT NULL,
		not_before TIMESTAMP NOT NULL,
		not_after TIMESTAMP NOT NULL,
		revoked_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY(ca_id) REFERENCES ca_config(ca_id)
	);`,
	`CREATE TABLE IF NOT EXISTS revoked_certificates (
		serial TEXT PRIMARY KEY,
		reason TEXT NOT NULL,
		revoked_at TIMESTAMP NOT NULL,
		FOREIGN KEY(serial) REFERENCES issued_certificates(serial)
	);`,
	`CREATE TABLE IF NOT EXISTS audit_logs (
		id TEXT PRIMARY KEY,
		action TEXT NOT NULL,
		actor TEXT,
		target TEXT,
		metadata TEXT,
		trace_id TEXT,
		created_at TIMESTAMP NOT NULL
	);`,
}

func (s *SQLStore) Migrate(ctx context.Context) error {
	for _, migration := range migrations {
		if _, err := s.db.ExecContext(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}
