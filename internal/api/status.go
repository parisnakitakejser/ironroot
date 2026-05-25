package api

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/db"
)

type statusOverviewResponse struct {
	Server       serverStatus       `json:"server"`
	Certificates certificateSummary `json:"certificates"`
	Enrollments  enrollmentSummary  `json:"enrollments"`
	Security     securitySummary    `json:"security"`
	Telemetry    telemetryStatus    `json:"telemetry"`
	CA           caStatus           `json:"ca"`
}

type serverStatus struct {
	Status       string `json:"status"`
	Version      string `json:"version"`
	APIHealth    string `json:"api_health"`
	Readiness    string `json:"readiness"`
	DatabaseType string `json:"database_type"`
	APIAddress   string `json:"api_address"`
	TLSEnabled   bool   `json:"tls_enabled"`
	GoVersion    string `json:"go_version"`
	Goroutines   int    `json:"goroutines"`
}

type certificateSummary struct {
	Active       int `json:"active"`
	ExpiringSoon int `json:"expiring_soon"`
	IssuedToday  int `json:"issued_today"`
	RenewedToday int `json:"renewed_today"`
	RevokedToday int `json:"revoked_today"`
	Revoked      int `json:"revoked"`
	Expired      int `json:"expired"`
	Total        int `json:"total"`
}

type enrollmentSummary struct {
	Successful           int `json:"successful"`
	Failed               int `json:"failed"`
	Pending              int `json:"pending"`
	InvalidTokenAttempts int `json:"invalid_token_attempts"`
	Total                int `json:"total"`
}

type securitySummary struct {
	Status           string `json:"status"`
	CriticalFindings int    `json:"critical_findings"`
	HighFindings     int    `json:"high_findings"`
	Warnings         int    `json:"warnings"`
	Message          string `json:"message"`
}

type telemetryStatus struct {
	Enabled           bool   `json:"enabled"`
	TracesEnabled     bool   `json:"traces_enabled"`
	MetricsEnabled    bool   `json:"metrics_enabled"`
	LogsEnabled       bool   `json:"logs_enabled"`
	OTLPEndpoint      string `json:"otlp_endpoint"`
	OTLPProtocol      string `json:"otlp_protocol"`
	PrometheusEnabled bool   `json:"prometheus_enabled"`
	PrometheusPath    string `json:"prometheus_path"`
	ExporterStatus    string `json:"exporter_status"`
}

type caStatus struct {
	RootFingerprint         string    `json:"root_fingerprint"`
	RootExpiresAt           time.Time `json:"root_expires_at,omitempty"`
	IntermediateFingerprint string    `json:"intermediate_fingerprint"`
	IntermediateExpiresAt   time.Time `json:"intermediate_expires_at,omitempty"`
	ActiveIssuer            string    `json:"active_issuer"`
	DisabledIssuers         int       `json:"disabled_issuers"`
	RetiredIssuers          int       `json:"retired_issuers"`
	ChainStatus             string    `json:"chain_status"`
	Warnings                []string  `json:"warnings"`
}

type caHierarchyStatus struct {
	Roots          []rootCAStatus     `json:"roots"`
	Summary        caHierarchySummary `json:"summary"`
	Warnings       []string           `json:"warnings,omitempty"`
	LegacyFallback bool               `json:"legacy_fallback"`
}

type caHierarchySummary struct {
	RootCAs          int `json:"root_cas"`
	IntermediateCAs  int `json:"intermediate_cas"`
	ActiveIssuers    int `json:"active_issuers"`
	DisabledIssuers  int `json:"disabled_issuers"`
	RetiredIssuers   int `json:"retired_issuers"`
	TokenPolicies    int `json:"token_policies"`
	Roles            int `json:"roles"`
	PendingApprovals int `json:"pending_approvals"`
}

type rootCAStatus struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Environment   string                 `json:"environment"`
	Fingerprint   string                 `json:"fingerprint"`
	Status        string                 `json:"status"`
	TrustDomain   string                 `json:"trust_domain,omitempty"`
	NotBefore     time.Time              `json:"not_before"`
	NotAfter      time.Time              `json:"not_after"`
	Intermediates []intermediateCAStatus `json:"intermediates"`
}

type intermediateCAStatus struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Environment     string                `json:"environment"`
	Owner           string                `json:"owner,omitempty"`
	Namespace       string                `json:"namespace,omitempty"`
	Fingerprint     string                `json:"fingerprint"`
	Status          string                `json:"status"`
	MaxTTL          string                `json:"max_ttl"`
	AllowedDNS      []string              `json:"allowed_dns,omitempty"`
	AllowedUsages   []string              `json:"allowed_usages,omitempty"`
	RequireApproval bool                  `json:"require_approval"`
	IssuanceLimit   int                   `json:"issuance_limit"`
	RenewalAllowed  bool                  `json:"renewal_allowed"`
	NotBefore       time.Time             `json:"not_before"`
	NotAfter        time.Time             `json:"not_after"`
	ActiveCerts     int                   `json:"active_certs"`
	RevokedCerts    int                   `json:"revoked_certs"`
	Roles           []caRoleStatus        `json:"roles,omitempty"`
	TokenPolicies   []caTokenPolicyStatus `json:"token_policies,omitempty"`
}

type caRoleStatus struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Subject     string   `json:"subject"`
	Permissions []string `json:"permissions"`
}

type caTokenPolicyStatus struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	CertificateTypes []string  `json:"certificate_types"`
	AllowedDNS       []string  `json:"allowed_dns,omitempty"`
	MaxTTL           string    `json:"max_ttl"`
	IssuanceLimit    int       `json:"issuance_limit"`
	RenewalAllowed   bool      `json:"renewal_allowed"`
	RequireApproval  bool      `json:"require_approval"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type certificateStatusItem struct {
	Serial        string    `json:"serial"`
	DNSNames      []string  `json:"dns_names"`
	Issuer        string    `json:"issuer"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	DaysRemaining int       `json:"days_remaining"`
	EnrollmentID  string    `json:"enrollment_id"`
	RenewalStatus string    `json:"renewal_status"`
}

type enrollmentStatusItem struct {
	ID               string    `json:"id"`
	Hostname         string    `json:"hostname"`
	MachineID        string    `json:"machine_id"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	LastSeenAt       time.Time `json:"last_seen_at,omitempty"`
	CertificateCount int       `json:"certificate_count"`
	FailedAttempts   int       `json:"failed_attempts"`
}

type tokenStatusItem struct {
	ID         string     `json:"id"`
	Host       string     `json:"host"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	Used       bool       `json:"used"`
	Revoked    bool       `json:"revoked"`
	UsageCount int        `json:"usage_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type auditStatusItem struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Result    string    `json:"result"`
	Severity  string    `json:"severity"`
}

func (h handler) statusOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	certs, _ := h.dep.Store.ListIssuedCertificates(ctx)
	enrollments, _ := h.dep.Store.ListEnrollments(ctx)
	auditLogs, _ := h.dep.Store.ListAuditLogs(ctx, 100)
	writeJSON(w, http.StatusOK, statusOverviewResponse{
		Server:       h.serverStatus(),
		Certificates: summarizeCertificates(certs, h.dep.Config.PKI.RenewBefore),
		Enrollments:  summarizeEnrollments(enrollments, auditLogs),
		Security:     summarizeSecurity(auditLogs),
		Telemetry:    h.telemetryStatus(),
		CA:           h.caStatus(),
	})
}

func (h handler) statusServer(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.serverStatus())
}

func (h handler) statusCA(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.caStatus())
}

func (h handler) statusCAHierarchy(w http.ResponseWriter, r *http.Request) {
	hierarchy, err := h.caHierarchyStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, hierarchy)
}

func (h handler) statusCertificates(w http.ResponseWriter, r *http.Request) {
	certs, err := h.dep.Store.ListIssuedCertificates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]certificateStatusItem, 0, len(certs))
	for _, cert := range certs {
		items = append(items, certificateItem(cert, h.dep.Config.PKI.RenewBefore))
	}
	writeJSON(w, http.StatusOK, items)
}

func (h handler) statusEnrollments(w http.ResponseWriter, r *http.Request) {
	enrollments, err := h.dep.Store.ListEnrollments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	certs, _ := h.dep.Store.ListIssuedCertificates(r.Context())
	counts := map[string]int{}
	lastSeen := map[string]time.Time{}
	for _, cert := range certs {
		counts[cert.EnrollmentID]++
		if cert.CreatedAt.After(lastSeen[cert.EnrollmentID]) {
			lastSeen[cert.EnrollmentID] = cert.CreatedAt
		}
	}
	items := make([]enrollmentStatusItem, 0, len(enrollments))
	for _, en := range enrollments {
		status := "pending"
		if counts[en.ID] > 0 {
			status = "active"
		}
		items = append(items, enrollmentStatusItem{ID: en.ID, Hostname: en.Hostname, MachineID: en.MachineID, Status: status, CreatedAt: en.CreatedAt, LastSeenAt: lastSeen[en.ID], CertificateCount: counts[en.ID]})
	}
	writeJSON(w, http.StatusOK, items)
}

func (h handler) statusTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := h.dep.Store.ListBootstrapTokens(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	now := time.Now().UTC()
	items := make([]tokenStatusItem, 0, len(tokens))
	for _, token := range tokens {
		usage, _ := h.dep.Store.BootstrapTokenUsage(r.Context(), token.ID)
		status := "active"
		if token.RevokedAt != nil {
			status = "revoked"
		} else if now.After(token.ExpiresAt) {
			status = "expired"
		} else if usage.UsageCount > 0 {
			status = "used"
		}
		items = append(items, tokenStatusItem{ID: token.ID, Host: token.Hostname, Status: status, CreatedAt: token.CreatedAt, ExpiresAt: token.ExpiresAt, Used: usage.UsageCount > 0, Revoked: token.RevokedAt != nil, UsageCount: usage.UsageCount, LastUsedAt: usage.LastUsedAt})
	}
	writeJSON(w, http.StatusOK, items)
}

func (h handler) statusSecurity(w http.ResponseWriter, r *http.Request) {
	logs, _ := h.dep.Store.ListAuditLogs(r.Context(), 100)
	writeJSON(w, http.StatusOK, summarizeSecurity(logs))
}

func (h handler) statusTelemetry(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.telemetryStatus())
}

func (h handler) auditRecent(w http.ResponseWriter, r *http.Request) {
	logs, err := h.dep.Store.ListAuditLogs(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]auditStatusItem, 0, len(logs))
	for _, log := range logs {
		items = append(items, auditStatusItem{Timestamp: log.CreatedAt, Actor: log.Actor, Action: log.Action, Target: log.Target, Result: auditResult(log), Severity: auditSeverity(log)})
	}
	writeJSON(w, http.StatusOK, items)
}

func (h handler) serverStatus() serverStatus {
	return serverStatus{Status: "healthy", Version: h.dep.Config.Telemetry.ServiceVersion, APIHealth: "ok", Readiness: "ready", DatabaseType: h.dep.Config.Database.Driver, APIAddress: h.dep.Config.Server.Address, TLSEnabled: h.dep.Config.Server.TLS.CertFile != "" && h.dep.Config.Server.TLS.KeyFile != "", GoVersion: runtime.Version(), Goroutines: runtime.NumGoroutine()}
}

func (h handler) telemetryStatus() telemetryStatus {
	cfg := h.dep.Config.Telemetry
	status := "disabled"
	if cfg.Enabled {
		status = "configured"
	}
	return telemetryStatus{Enabled: cfg.Enabled, TracesEnabled: cfg.Traces.Enabled, MetricsEnabled: cfg.Metrics.Enabled, LogsEnabled: cfg.Logs.Enabled, OTLPEndpoint: cfg.Exporter.Endpoint, OTLPProtocol: cfg.Exporter.Protocol, PrometheusEnabled: cfg.Prometheus.Enabled, PrometheusPath: cfg.Prometheus.Path, ExporterStatus: status}
}

func (h handler) caStatus() caStatus {
	if h.dep.Authority == nil {
		return caStatus{
			ActiveIssuer: "unconfigured",
			ChainStatus:  "unconfigured",
			Warnings:     []string{"CA authority is not configured"},
		}
	}
	rootCerts := parsePEMCertificates(h.dep.Authority.RootPEM())
	chainCerts := parsePEMCertificates(h.dep.Authority.ChainPEM())
	status := caStatus{ActiveIssuer: h.dep.Authority.CAID(), ChainStatus: "unknown"}
	if len(rootCerts) > 0 {
		status.RootFingerprint = fingerprint(rootCerts[0])
		status.RootExpiresAt = rootCerts[0].NotAfter
		if time.Until(rootCerts[0].NotAfter) < 2*365*24*time.Hour {
			status.Warnings = append(status.Warnings, "Root CA expires within 2 years")
		}
	}
	if len(chainCerts) > 0 {
		intermediate := chainCerts[0]
		status.IntermediateFingerprint = fingerprint(intermediate)
		status.IntermediateExpiresAt = intermediate.NotAfter
		status.ChainStatus = "valid"
		if time.Until(intermediate.NotAfter) < 180*24*time.Hour {
			status.Warnings = append(status.Warnings, "Intermediate CA expires within 180 days")
		}
	}
	if len(chainCerts) == 0 && len(rootCerts) == 0 {
		status.ChainStatus = "unconfigured"
		status.Warnings = append(status.Warnings, "CA material is not configured")
	}
	return status
}

func (h handler) caHierarchyStatus(ctx context.Context) (caHierarchyStatus, error) {
	roots, err := h.dep.Store.ListRootCAs(ctx)
	if err != nil {
		return caHierarchyStatus{}, err
	}
	intermediates, err := h.dep.Store.ListIntermediateCAs(ctx)
	if err != nil {
		return caHierarchyStatus{}, err
	}
	roles, err := h.dep.Store.ListCARoles(ctx)
	if err != nil {
		return caHierarchyStatus{}, err
	}
	policies, err := h.dep.Store.ListCATokenPolicies(ctx)
	if err != nil {
		return caHierarchyStatus{}, err
	}
	certs, _ := h.dep.Store.ListIssuedCertificates(ctx)
	if len(roots) == 0 && len(intermediates) == 0 {
		return h.legacyCAHierarchy(ctx, certs)
	}
	return buildCAHierarchy(roots, intermediates, roles, policies, certs), nil
}

func (h handler) legacyCAHierarchy(ctx context.Context, certs []db.IssuedCertificate) (caHierarchyStatus, error) {
	configs, err := h.dep.Store.ListCAConfigs(ctx)
	if err != nil || len(configs) == 0 {
		return caHierarchyStatus{
			Summary:  caHierarchySummary{},
			Warnings: []string{"No multi-root CA hierarchy metadata has been registered yet."},
		}, nil
	}
	roots := make([]db.RootCA, 0, len(configs))
	intermediates := make([]db.IntermediateCA, 0, len(configs))
	for _, cfg := range configs {
		rootID := "legacy-root-" + cfg.CAID
		roots = append(roots, db.RootCA{
			ID:          rootID,
			Name:        cfg.Name + " Root",
			Environment: "legacy",
			Fingerprint: cfg.RootFingerprint,
			Status:      cfg.Status,
			CreatedAt:   cfg.CreatedAt,
			NotBefore:   cfg.NotBefore,
			NotAfter:    cfg.NotAfter,
		})
		intermediates = append(intermediates, db.IntermediateCA{
			ID:             cfg.CAID,
			RootID:         rootID,
			Name:           cfg.Name,
			Environment:    "legacy",
			Fingerprint:    cfg.IntermediateFingerprint,
			Status:         cfg.Status,
			RenewalAllowed: true,
			CreatedAt:      cfg.CreatedAt,
			NotBefore:      cfg.NotBefore,
			NotAfter:       cfg.NotAfter,
		})
	}
	hierarchy := buildCAHierarchy(roots, intermediates, nil, nil, certs)
	hierarchy.LegacyFallback = true
	hierarchy.Warnings = append(hierarchy.Warnings, "Rendered legacy ca_config rows as a one-root-per-issuer hierarchy.")
	return hierarchy, nil
}

func buildCAHierarchy(roots []db.RootCA, intermediates []db.IntermediateCA, roles []db.CARole, policies []db.CATokenPolicy, certs []db.IssuedCertificate) caHierarchyStatus {
	rootItems := make([]rootCAStatus, 0, len(roots))
	rootIndex := map[string]int{}
	for _, root := range roots {
		rootIndex[root.ID] = len(rootItems)
		rootItems = append(rootItems, rootCAStatus{
			ID:            root.ID,
			Name:          root.Name,
			Environment:   root.Environment,
			Fingerprint:   root.Fingerprint,
			Status:        root.Status,
			TrustDomain:   root.TrustDomain,
			NotBefore:     root.NotBefore,
			NotAfter:      root.NotAfter,
			Intermediates: []intermediateCAStatus{},
		})
	}
	roleByIntermediate := map[string][]caRoleStatus{}
	for _, role := range roles {
		roleByIntermediate[role.IntermediateID] = append(roleByIntermediate[role.IntermediateID], caRoleStatus{ID: role.ID, Name: role.Name, Subject: role.Subject, Permissions: splitNames(role.Permissions)})
	}
	policyByIntermediate := map[string][]caTokenPolicyStatus{}
	for _, policy := range policies {
		policyByIntermediate[policy.IntermediateID] = append(policyByIntermediate[policy.IntermediateID], caTokenPolicyStatus{
			ID: policy.ID, Name: policy.Name, CertificateTypes: splitNames(policy.CertificateTypes), AllowedDNS: splitNames(policy.AllowedDNS),
			MaxTTL: policy.MaxTTL.String(), IssuanceLimit: policy.IssuanceLimit, RenewalAllowed: policy.RenewalAllowed, RequireApproval: policy.RequireApproval, ExpiresAt: policy.ExpiresAt,
		})
	}
	activeCerts, revokedCerts := certificateCountsByCA(certs)
	var summary caHierarchySummary
	summary.RootCAs = len(rootItems)
	summary.IntermediateCAs = len(intermediates)
	summary.TokenPolicies = len(policies)
	summary.Roles = len(roles)
	for _, intermediate := range intermediates {
		item := intermediateCAStatus{
			ID: intermediate.ID, Name: intermediate.Name, Environment: intermediate.Environment, Owner: intermediate.Owner, Namespace: intermediate.Namespace,
			Fingerprint: intermediate.Fingerprint, Status: intermediate.Status, MaxTTL: intermediate.MaxTTL.String(), AllowedDNS: splitNames(intermediate.AllowedDNS),
			AllowedUsages: splitNames(intermediate.AllowedUsages), RequireApproval: intermediate.RequireApproval, IssuanceLimit: intermediate.IssuanceLimit,
			RenewalAllowed: intermediate.RenewalAllowed, NotBefore: intermediate.NotBefore, NotAfter: intermediate.NotAfter, ActiveCerts: activeCerts[intermediate.ID],
			RevokedCerts: revokedCerts[intermediate.ID], Roles: roleByIntermediate[intermediate.ID], TokenPolicies: policyByIntermediate[intermediate.ID],
		}
		switch intermediate.Status {
		case "active":
			summary.ActiveIssuers++
		case "disabled":
			summary.DisabledIssuers++
		case "retired":
			summary.RetiredIssuers++
		}
		if intermediate.RequireApproval {
			summary.PendingApprovals++
		}
		if idx, ok := rootIndex[intermediate.RootID]; ok {
			rootItems[idx].Intermediates = append(rootItems[idx].Intermediates, item)
		}
	}
	return caHierarchyStatus{Roots: rootItems, Summary: summary}
}

func certificateCountsByCA(certs []db.IssuedCertificate) (map[string]int, map[string]int) {
	active := map[string]int{}
	revoked := map[string]int{}
	now := time.Now().UTC()
	for _, cert := range certs {
		if cert.RevokedAt != nil || now.After(cert.NotAfter) {
			revoked[cert.CAID]++
			continue
		}
		active[cert.CAID]++
	}
	return active, revoked
}

func summarizeCertificates(certs []db.IssuedCertificate, renewBefore time.Duration) certificateSummary {
	var out certificateSummary
	now := time.Now().UTC()
	today := startOfDay(now)
	out.Total = len(certs)
	for _, cert := range certs {
		if cert.CreatedAt.After(today) {
			out.IssuedToday++
		}
		if cert.RevokedAt != nil {
			out.Revoked++
			if cert.RevokedAt.After(today) {
				out.RevokedToday++
			}
			continue
		}
		if now.After(cert.NotAfter) {
			out.Expired++
			continue
		}
		out.Active++
		if time.Until(cert.NotAfter) <= renewBefore {
			out.ExpiringSoon++
		}
	}
	return out
}

func summarizeEnrollments(enrollments []db.Enrollment, logs []db.AuditLog) enrollmentSummary {
	var out enrollmentSummary
	out.Total = len(enrollments)
	out.Successful = len(enrollments)
	for _, log := range logs {
		if strings.Contains(log.Action, "enrollment") && strings.Contains(log.Action, "failed") {
			out.Failed++
		}
		if strings.Contains(log.Action, "token") && strings.Contains(log.Action, "invalid") {
			out.InvalidTokenAttempts++
		}
	}
	return out
}

func summarizeSecurity(logs []db.AuditLog) securitySummary {
	out := securitySummary{Status: "unknown", Message: "No security-check result has been recorded yet."}
	for _, log := range logs {
		if strings.Contains(log.Action, "security-check") || strings.Contains(log.Action, "security_check") {
			out.Status = "warning"
			out.Warnings++
			out.Message = "Recent security-check activity found in audit log."
			if strings.Contains(strings.ToLower(log.Metadata), "critical") {
				out.CriticalFindings++
				out.Status = "critical"
			}
			if strings.Contains(strings.ToLower(log.Metadata), "high") {
				out.HighFindings++
				if out.Status != "critical" {
					out.Status = "high"
				}
			}
		}
	}
	return out
}

func certificateItem(cert db.IssuedCertificate, renewBefore time.Duration) certificateStatusItem {
	now := time.Now().UTC()
	status := "active"
	if cert.RevokedAt != nil {
		status = "revoked"
	} else if now.After(cert.NotAfter) {
		status = "expired"
	}
	renewal := "ok"
	if cert.RevokedAt == nil && now.Before(cert.NotAfter) && time.Until(cert.NotAfter) <= renewBefore {
		renewal = "due"
	}
	return certificateStatusItem{Serial: cert.Serial, DNSNames: splitNames(cert.DNSNames), Issuer: cert.CAID, Status: status, CreatedAt: cert.CreatedAt, ExpiresAt: cert.NotAfter, DaysRemaining: int(time.Until(cert.NotAfter).Hours() / 24), EnrollmentID: cert.EnrollmentID, RenewalStatus: renewal}
}

func splitNames(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func auditResult(log db.AuditLog) string {
	if strings.Contains(strings.ToLower(log.Action), "failed") {
		return "failure"
	}
	return "success"
}

func auditSeverity(log db.AuditLog) string {
	action := strings.ToLower(log.Action)
	metadata := strings.ToLower(log.Metadata)
	switch {
	case strings.Contains(metadata, "critical"):
		return "critical"
	case strings.Contains(metadata, "high"):
		return "high"
	case strings.Contains(action, "failed"), strings.Contains(action, "revoked"):
		return "warning"
	default:
		return "info"
	}
}

func parsePEMCertificates(in string) []*x509.Certificate {
	var certs []*x509.Certificate
	rest := []byte(in)
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err == nil {
			certs = append(certs, cert)
		}
	}
	return certs
}

func fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
