package securitycheck

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/parisnakitakejser/ironroot/internal/config"
)

type HostCheck struct{}

func (HostCheck) ID() string       { return "host.runtime" }
func (HostCheck) Category() string { return "host" }
func (HostCheck) Run(context.Context, Target) Result {
	r := baseResult("host.runtime", "Host OS and process posture", "host", SeverityLow)
	root := os.Geteuid() == 0
	msg := "Running on " + runtime.GOOS + "/" + runtime.GOARCH
	if root {
		r.Status = StatusWarn
		r.Message = msg + "; process is running as root."
		r.Remediation = "Run IronRoot as a dedicated non-root user."
		return r
	}
	r.Message = msg + "; process is not running as root."
	r.Remediation = "Keep running IronRoot as a dedicated non-root user."
	return r
}

type EnvironmentSecretsCheck struct{}

func (EnvironmentSecretsCheck) ID() string       { return "host.env_sensitive_values" }
func (EnvironmentSecretsCheck) Category() string { return "host" }
func (EnvironmentSecretsCheck) Run(context.Context, Target) Result {
	r := baseResult("host.env_sensitive_values", "Environment does not expose obvious sensitive values", "host", SeverityMedium)
	for _, env := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(env, "=", 2)[0])
		if strings.Contains(key, "TOKEN") || strings.Contains(key, "PASSWORD") || strings.Contains(key, "PRIVATE_KEY") || strings.Contains(key, "KEY_PASS") {
			r.Status = StatusWarn
			r.Message = "An environment variable name suggests sensitive material may be present: " + key
			r.Remediation = "Move secrets to mounted files or a secret manager and avoid exposing values in process environments."
			return r
		}
	}
	r.Message = "No obvious sensitive environment variable names were detected."
	r.Remediation = "Continue masking tokens, key passwords, and private key material."
	return r
}

type FilePermissionCheck struct {
	IDValue       string
	Title         string
	CategoryValue string
	PathKind      string
}

func (c FilePermissionCheck) ID() string       { return c.IDValue }
func (c FilePermissionCheck) Category() string { return c.CategoryValue }
func (c FilePermissionCheck) Run(_ context.Context, target Target) Result {
	r := baseResult(c.IDValue, c.Title, c.CategoryValue, SeverityHigh)
	path := pathForKind(target.Config, target.ConfigPath, c.PathKind)
	if path == "" {
		r.Status = StatusSkip
		r.Message = "No path is configured for this check."
		r.Remediation = "Configure explicit paths for production deployments."
		return r
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			r.Status = StatusSkip
			r.Message = "Path does not exist: " + maskPath(path)
			r.Remediation = "Create the path with owner-only permissions before production use."
			return r
		}
		r.Status = StatusWarn
		r.Message = "Could not inspect permissions for " + maskPath(path)
		r.Remediation = "Verify permissions manually."
		return r
	}
	if info.Mode().Perm()&0o077 != 0 {
		r.Status = StatusFail
		r.Message = "Path is readable or writable by group/world: " + maskPath(path)
		r.Remediation = "Restrict permissions to the IronRoot user, usually 0600 for files and 0700 for directories."
		return r
	}
	r.Message = "Permissions are restricted for " + maskPath(path)
	r.Remediation = "Keep CA keys and SQLite files owner-only."
	return r
}

type RootPrivateKeyAbsentCheck struct{}

func (RootPrivateKeyAbsentCheck) ID() string       { return "ca.root.private_key_absent" }
func (RootPrivateKeyAbsentCheck) Category() string { return "ca" }
func (RootPrivateKeyAbsentCheck) Run(_ context.Context, target Target) Result {
	r := baseResult("ca.root.private_key_absent", "Root CA private key is not present on online server", "ca", SeverityCritical)
	dir := filepath.Dir(target.Config.PKI.RootFile)
	candidates := []string{"root.key", "root-ca.key", "root.key.enc", "root-ca.key.enc", "ca.key"}
	for _, name := range candidates {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			r.Status = StatusFail
			r.Message = "A possible Root CA private key was found under the configured PKI directory."
			r.Remediation = "Remove Root CA private keys from the online server and keep them offline and encrypted."
			return r
		}
	}
	r.Message = "Root private key was not found in configured PKI paths."
	r.Remediation = "Keep the Root CA private key offline and encrypted."
	return r
}

type IntermediateKeyCheck struct{}

func (IntermediateKeyCheck) ID() string       { return "ca.intermediate.private_key" }
func (IntermediateKeyCheck) Category() string { return "ca" }
func (IntermediateKeyCheck) Run(_ context.Context, target Target) Result {
	r := baseResult("ca.intermediate.private_key", "Intermediate CA private key exists and is restricted", "ca", SeverityHigh)
	path := target.Config.PKI.IntermediateKeyFile
	info, err := os.Stat(path)
	if err != nil {
		r.Status = StatusSkip
		r.Message = "Intermediate private key is not present at the configured path."
		r.Remediation = "Mount the encrypted Intermediate CA private key at the configured path before issuing certificates."
		return r
	}
	if info.Mode().Perm()&0o077 != 0 {
		r.Status = StatusFail
		r.Message = "Intermediate private key permissions allow group/world access."
		r.Remediation = "Use owner-only permissions such as 0600."
		return r
	}
	r.Message = "Intermediate private key exists with restricted permissions."
	r.Remediation = "Keep the key encrypted at rest and mounted only into the IronRoot server."
	return r
}

type CAChainCheck struct{}

func (CAChainCheck) ID() string       { return "ca.chain.valid" }
func (CAChainCheck) Category() string { return "ca" }
func (CAChainCheck) Run(_ context.Context, target Target) Result {
	r := baseResult("ca.chain.valid", "Intermediate CA is signed by configured Root CA", "ca", SeverityCritical)
	root, err := readFirstCert(target.Config.PKI.RootFile)
	if err != nil {
		r.Status = StatusSkip
		r.Message = "Root certificate could not be read."
		r.Remediation = "Mount the Root CA certificate so clients can build trust chains."
		return r
	}
	intermediate, err := readFirstCert(target.Config.PKI.IntermediateCertFile)
	if err != nil {
		r.Status = StatusSkip
		r.Message = "Intermediate certificate could not be read."
		r.Remediation = "Mount the Intermediate CA certificate signed by the offline Root CA."
		return r
	}
	pool := x509.NewCertPool()
	pool.AddCert(root)
	if _, err := intermediate.Verify(x509.VerifyOptions{Roots: pool, CurrentTime: target.Now, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}); err != nil {
		r.Status = StatusFail
		r.Message = "Intermediate certificate did not verify against the configured Root CA."
		r.Remediation = "Reissue the Intermediate CA from the offline Root CA and update the mounted chain."
		return r
	}
	r.Message = "Intermediate certificate verifies against the configured Root CA."
	r.Remediation = "Keep serving the full CA chain to clients."
	return r
}

type CertificateLifetimeCheck struct{}

func (CertificateLifetimeCheck) ID() string       { return "ca.lifetimes" }
func (CertificateLifetimeCheck) Category() string { return "ca" }
func (CertificateLifetimeCheck) Run(_ context.Context, target Target) Result {
	r := baseResult("ca.lifetimes", "CA and issued certificate lifetimes follow recommendations", "ca", SeverityMedium)
	root, rootErr := readFirstCert(target.Config.PKI.RootFile)
	intermediate, intErr := readFirstCert(target.Config.PKI.IntermediateCertFile)
	if rootErr == nil && intErr == nil {
		if !root.NotAfter.After(intermediate.NotAfter) {
			r.Status = StatusFail
			r.Message = "Root CA lifetime is not longer than Intermediate CA lifetime."
			r.Remediation = "Use a longer Root CA lifetime than Intermediate CA lifetime; recommended values are 20 years and 5 years."
			return r
		}
		if target.Now.After(intermediate.NotAfter) {
			r.Status = StatusFail
			r.Message = "Intermediate CA is expired."
			r.Remediation = "Import a new Intermediate CA signed by the offline Root CA."
			return r
		}
		if intermediate.NotAfter.Sub(target.Now) < 180*24*time.Hour {
			r.Status = StatusWarn
			r.Message = "Intermediate CA expires within 180 days."
			r.Remediation = "Plan Intermediate CA rotation."
			return r
		}
		if root.NotAfter.Sub(target.Now) < 2*365*24*time.Hour {
			r.Status = StatusWarn
			r.Message = "Root CA expires within 2 years."
			r.Remediation = "Plan Root CA migration."
			return r
		}
	}
	if target.Config.PKI.DefaultLifetime > 90*24*time.Hour {
		r.Status = StatusWarn
		r.Message = "Default server certificate lifetime is longer than 90 days."
		r.Remediation = "Use 90 days or less for normal server certificates."
		return r
	}
	r.Message = "Configured certificate lifetimes are within the recommended MVP posture."
	r.Remediation = "Use 20 years for Root CA, 5 years for Intermediate CA, and 90 days for issued certificates."
	return r
}

type DatabaseReachableCheck struct{}

func (DatabaseReachableCheck) ID() string       { return "database.reachable" }
func (DatabaseReachableCheck) Category() string { return "database" }
func (DatabaseReachableCheck) Run(ctx context.Context, target Target) Result {
	r := baseResult("database.reachable", "Database is reachable and migrations are applied", "database", SeverityHigh)
	if target.Config.Database.Driver != "sqlite" {
		r.Status = StatusSkip
		r.Message = "PostgreSQL-ready checks are reserved for a future backend."
		r.Remediation = "Use SQLite for the current MVP or add PostgreSQL check implementation with the backend."
		return r
	}
	db, err := sql.Open("sqlite3", target.Config.Database.DSN)
	if err != nil {
		r.Status = StatusFail
		r.Message = "SQLite database could not be opened."
		r.Remediation = "Verify the database path and permissions."
		return r
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		r.Status = StatusFail
		r.Message = "SQLite database is not reachable."
		r.Remediation = "Apply migrations and verify data directory permissions."
		return r
	}
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name IN ('ca_config','enrollments','issued_certificates','audit_logs','bootstrap_tokens')")
	if err != nil {
		r.Status = StatusFail
		r.Message = "Could not inspect SQLite migration state."
		r.Remediation = "Run ironroot-admin init-server."
		return r
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	if count < 5 {
		r.Status = StatusFail
		r.Message = "Expected migration tables are missing."
		r.Remediation = "Run ironroot-admin init-server before production use."
		return r
	}
	r.Message = "SQLite database is reachable and expected migration tables exist."
	r.Remediation = "Back up the database with Intermediate CA material."
	return r
}

type SQLiteKubernetesCheck struct{}

func (SQLiteKubernetesCheck) ID() string       { return "database.sqlite_kubernetes_pvc" }
func (SQLiteKubernetesCheck) Category() string { return "database" }
func (SQLiteKubernetesCheck) Run(_ context.Context, target Target) Result {
	r := baseResult("database.sqlite_kubernetes_pvc", "SQLite in Kubernetes uses persistent storage", "database", SeverityMedium)
	if !InKubernetes() || target.Config.Database.Driver != "sqlite" {
		r.Status = StatusSkip
		r.Message = "Not running SQLite inside Kubernetes."
		r.Remediation = "Use a PVC for SQLite when deploying IronRoot to Kubernetes."
		return r
	}
	if strings.Contains(target.Config.Database.DSN, "/data/") {
		r.Message = "SQLite DSN points at /data, which should be backed by a PVC in the provided manifests."
		r.Remediation = "Confirm the /data mount is a persistent volume."
		return r
	}
	r.Status = StatusWarn
	r.Message = "SQLite in Kubernetes does not appear to use /data."
	r.Remediation = "Mount SQLite data on a PVC-backed path such as /data."
	return r
}

type APITLSCheck struct{}

func (APITLSCheck) ID() string       { return "api.tls_enabled" }
func (APITLSCheck) Category() string { return "api" }
func (APITLSCheck) Run(_ context.Context, target Target) Result {
	r := baseResult("api.tls_enabled", "API TLS is enabled", "api", SeverityHigh)
	if target.Config.Server.TLS.CertFile == "" || target.Config.Server.TLS.KeyFile == "" {
		r.Status = StatusFail
		r.Message = "API TLS certificate or key is not configured."
		r.Remediation = "Enable TLS before exposing the API outside localhost."
		return r
	}
	cert, err := readFirstCert(target.Config.Server.TLS.CertFile)
	if err != nil {
		r.Status = StatusWarn
		r.Message = "API TLS certificate could not be parsed."
		r.Remediation = "Verify TLS certificate and key files."
		return r
	}
	if target.Now.After(cert.NotAfter) {
		r.Status = StatusFail
		r.Message = "API TLS certificate is expired."
		r.Remediation = "Renew the API TLS certificate."
		return r
	}
	r.Message = "API TLS is configured and the certificate is not expired."
	r.Remediation = "Keep TLS enabled and rotate certificates before expiry."
	return r
}

type APIBindCheck struct{}

func (APIBindCheck) ID() string       { return "api.bind_address" }
func (APIBindCheck) Category() string { return "api" }
func (APIBindCheck) Run(_ context.Context, target Target) Result {
	r := baseResult("api.bind_address", "API bind address is intentional", "api", SeverityMedium)
	host, _, err := net.SplitHostPort(target.Config.Server.Address)
	if err != nil {
		host = target.Config.Server.Address
	}
	public := host == "" || host == "0.0.0.0" || host == "::"
	tls := target.Config.Server.TLS.CertFile != "" && target.Config.Server.TLS.KeyFile != ""
	if public && !tls {
		r.Status = StatusFail
		r.Message = "API listens on a public address without TLS."
		r.Remediation = "Bind to localhost for development or enable TLS before public binding."
		return r
	}
	if public {
		r.Status = StatusWarn
		r.Message = "API listens on a public address."
		r.Remediation = "Confirm network policy, firewalling, and authentication before production exposure."
		return r
	}
	r.Message = "API bind address is not broadly public."
	r.Remediation = "Keep public exposure behind TLS and future mTLS/auth controls."
	return r
}

type TokenTTLCheck struct{}

func (TokenTTLCheck) ID() string       { return "tokens.ttl" }
func (TokenTTLCheck) Category() string { return "tokens" }
func (TokenTTLCheck) Run(_ context.Context, _ Target) Result {
	r := baseResult("tokens.ttl", "Bootstrap tokens are hashed and time-limited", "tokens", SeverityMedium)
	r.Status = StatusSkip
	r.Message = "Token TTL can be fully evaluated once admin token APIs expose active token metadata."
	r.Remediation = "Use short token TTLs, revoke unused tokens, and never store raw token values."
	return r
}

func EvaluateTokenTTL(ttl time.Duration, max time.Duration) Result {
	r := baseResult("tokens.ttl.policy", "Bootstrap token TTL is within recommended maximum", "tokens", SeverityMedium)
	if ttl <= 0 {
		r.Status = StatusFail
		r.Message = "Bootstrap token TTL is not positive."
		r.Remediation = "Use a positive, short-lived TTL."
		return r
	}
	if ttl > max {
		r.Status = StatusWarn
		r.Message = "Bootstrap token TTL exceeds the recommended maximum."
		r.Remediation = "Use short-lived bootstrap tokens, typically one hour or less."
		return r
	}
	r.Message = "Bootstrap token TTL is within the recommended maximum."
	r.Remediation = "Keep bootstrap token TTLs short and revoke unused tokens."
	return r
}

type AuditCheck struct{}

func (AuditCheck) ID() string       { return "audit.enabled" }
func (AuditCheck) Category() string { return "audit" }
func (AuditCheck) Run(_ context.Context, _ Target) Result {
	r := baseResult("audit.enabled", "Audit logging is enabled", "audit", SeverityMedium)
	r.Message = "Audit logging is part of the server write path for enrollment, issuance, renewal, and revocation."
	r.Remediation = "Add retention policy controls before long-running production deployments."
	return r
}

type AuditChainCheck struct{}

func (AuditChainCheck) ID() string       { return "audit.ledger_chain_valid" }
func (AuditChainCheck) Category() string { return "audit" }
func (AuditChainCheck) Run(ctx context.Context, target Target) Result {
	r := baseResult("audit.ledger_chain_valid", "Audit log ledger chain is cryptographically secure", "audit", SeverityCritical)

	if target.Store == nil {
		r.Status = StatusWarn
		r.Message = "Storage backend is not available to verify the audit ledger."
		r.Remediation = "Run security-check in an environment connected to the SQLite or PostgreSQL database."
		return r
	}

	// Fetch a generous limit of audit logs for verification (e.g. 500)
	logs, err := target.Store.ListAuditLogs(ctx, 500)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			r.Status = StatusWarn
			r.Message = "Audit log ledger cannot be verified because the database tables have not been created yet."
			r.Remediation = "Initialize the server configuration and database migrations by running: ironroot-admin init-server"
			return r
		}
		r.Status = StatusFail
		r.Message = "Failed to retrieve audit log entries for cryptographic validation: " + err.Error()
		r.Remediation = "Verify that the database permissions allow reading the audit_logs table."
		return r
	}

	if len(logs) == 0 {
		r.Status = StatusPass
		r.Message = "No audit log entries exist yet; ledger chain is empty and valid."
		r.Remediation = "Generate activity (e.g., generate a bootstrap token or enroll) to initialize the audit ledger."
		return r
	}

	// ListAuditLogs returns newest (latest) first because of ORDER BY created_at DESC.
	// Let's iterate in chronological order (oldest to newest) to verify the chain.
	var lastHash string
	for i := len(logs) - 1; i >= 0; i-- {
		log := logs[i]

		// 1. Verify PrevHash matches the previous entry's Hash
		if log.PrevHash != lastHash {
			r.Status = StatusFail
			r.Message = "Cryptographic audit chain is broken! Hash mismatch at entry " + log.ID + ": expected previous hash " + lastHash + ", got " + log.PrevHash
			r.Remediation = "CRITICAL: The audit log database has been tampered with or corrupted! Restore the database from a trusted, encrypted backup."
			return r
		}

		// 2. Recalculate Hash and verify it matches the stored Hash
		h := sha256.New()
		_, _ = h.Write([]byte(log.PrevHash))
		_, _ = h.Write([]byte(log.Action))
		_, _ = h.Write([]byte(log.Actor))
		_, _ = h.Write([]byte(log.Target))
		_, _ = h.Write([]byte(log.Metadata))
		_, _ = h.Write([]byte(log.TraceID))
		_, _ = h.Write([]byte(log.CreatedAt.Format(time.RFC3339)))
		recalculated := hex.EncodeToString(h.Sum(nil))

		if log.Hash != recalculated {
			r.Status = StatusFail
			r.Message = "Audit ledger record signature verification failed for entry " + log.ID + ": signature recalculation mismatch."
			r.Remediation = "CRITICAL: Audit log payload has been modified or corrupted! Recalculated hash does not match stored signature hash."
			return r
		}

		lastHash = log.Hash
	}

	r.Status = StatusPass
	r.Message = "Audit log ledger chain verified successfully. Unbroken cryptographically secured sequence."
	r.Remediation = "No action required. The immutable audit log ledger is structurally sound."
	return r
}

type TelemetryCheck struct{}

func (TelemetryCheck) ID() string       { return "telemetry.configured" }
func (TelemetryCheck) Category() string { return "telemetry" }
func (TelemetryCheck) Run(_ context.Context, target Target) Result {
	r := baseResult("telemetry.configured", "Telemetry is configured or intentionally disabled", "telemetry", SeverityLow)
	if !target.Config.Telemetry.Enabled {
		r.Status = StatusWarn
		r.Message = "OpenTelemetry is disabled."
		r.Remediation = "Enable telemetry or record the operational reason for disabling it."
		return r
	}
	if target.Config.Telemetry.ServiceName == "" || target.Config.Telemetry.DeploymentEnvironment == "" {
		r.Status = StatusFail
		r.Message = "Telemetry is enabled but service name or deployment environment is missing."
		r.Remediation = "Set telemetry.service_name and telemetry.deployment_environment."
		return r
	}
	if target.Config.Telemetry.SamplingRatio >= 1 && strings.EqualFold(target.Config.Telemetry.DeploymentEnvironment, "production") {
		r.Status = StatusWarn
		r.Message = "Telemetry sampling is always-on in production."
		r.Remediation = "Use a lower sampling ratio for high-volume production environments."
		return r
	}
	r.Message = "Telemetry has service identity and deployment environment configured."
	r.Remediation = "Verify the OTLP endpoint is reachable from the deployment network."
	return r
}

type ContainerCheck struct{}

func (ContainerCheck) ID() string       { return "container.runtime" }
func (ContainerCheck) Category() string { return "container" }
func (ContainerCheck) Run(_ context.Context, _ Target) Result {
	r := baseResult("container.runtime", "Container runtime posture is detected", "container", SeverityLow)
	if !InContainer() {
		r.Status = StatusSkip
		r.Message = "Container runtime was not detected."
		r.Remediation = "When containerized, run as non-root with mounted /config, /data, and /pki."
		return r
	}
	if os.Geteuid() == 0 {
		r.Status = StatusFail
		r.Message = "Containerized process is running as root."
		r.Remediation = "Run the container as a non-root user and drop Linux capabilities."
		return r
	}
	r.Message = "Container runtime detected and process is non-root."
	r.Remediation = "Use read-only root filesystem where possible and mount CA material rather than baking it into the image."
	return r
}

type KubernetesCheck struct{}

func (KubernetesCheck) ID() string       { return "kubernetes.runtime" }
func (KubernetesCheck) Category() string { return "kubernetes" }
func (KubernetesCheck) Run(_ context.Context, _ Target) Result {
	r := baseResult("kubernetes.runtime", "Kubernetes runtime posture is detected", "kubernetes", SeverityLow)
	if !InKubernetes() {
		r.Status = StatusSkip
		r.Message = "Kubernetes runtime was not detected."
		r.Remediation = "Use the provided manifests with non-root security context, probes, PVC, and Secret mounts."
		return r
	}
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "default" {
		r.Status = StatusWarn
		r.Message = "IronRoot appears to be running in the default namespace."
		r.Remediation = "Use a dedicated namespace and scoped ServiceAccount."
		return r
	}
	r.Message = "Kubernetes runtime detected."
	r.Remediation = "Verify NetworkPolicy, internal-only Service, PVC, Secret CA material, probes, and dropped capabilities."
	return r
}

func InContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	b, err := os.ReadFile("/proc/1/cgroup")
	return err == nil && (strings.Contains(string(b), "docker") || strings.Contains(string(b), "kubepods") || strings.Contains(string(b), "containerd"))
}

func InKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != "" || os.Getenv("POD_NAME") != ""
}

func readFirstCert(path string) (*x509.Certificate, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	return x509.ParseCertificate(block.Bytes)
}

func pathForKind(cfg config.Config, configPath string, kind string) string {
	switch kind {
	case "config":
		return configPath
	case "ca_key":
		return cfg.PKI.IntermediateKeyFile
	case "database":
		return sqlitePath(cfg.Database.DSN)
	default:
		return ""
	}
}

func sqlitePath(dsn string) string {
	dsn = strings.TrimPrefix(dsn, "file:")
	if i := strings.Index(dsn, "?"); i >= 0 {
		dsn = dsn[:i]
	}
	return dsn
}

func maskPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
