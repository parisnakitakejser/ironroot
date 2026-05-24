package irtop

import "time"

type Snapshot struct {
	Overview     Overview      `json:"overview,omitempty"`
	Server       ServerStatus  `json:"server,omitempty"`
	CA           CAStatus      `json:"ca,omitempty"`
	Certificates []Certificate `json:"certificates,omitempty"`
	Enrollments  []Enrollment  `json:"enrollments,omitempty"`
	Tokens       []Token       `json:"tokens,omitempty"`
	Security     Security      `json:"security,omitempty"`
	Telemetry    Telemetry     `json:"telemetry,omitempty"`
	Audit        []AuditEvent  `json:"audit,omitempty"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type Overview struct {
	Server       ServerStatus       `json:"server"`
	Certificates CertificateSummary `json:"certificates"`
	Enrollments  EnrollmentSummary  `json:"enrollments"`
	Security     Security           `json:"security"`
	Telemetry    Telemetry          `json:"telemetry"`
	CA           CAStatus           `json:"ca"`
}

type ServerStatus struct {
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

type CertificateSummary struct {
	Active       int `json:"active"`
	ExpiringSoon int `json:"expiring_soon"`
	IssuedToday  int `json:"issued_today"`
	RenewedToday int `json:"renewed_today"`
	RevokedToday int `json:"revoked_today"`
	Revoked      int `json:"revoked"`
	Expired      int `json:"expired"`
	Total        int `json:"total"`
}

type EnrollmentSummary struct {
	Successful           int `json:"successful"`
	Failed               int `json:"failed"`
	Pending              int `json:"pending"`
	InvalidTokenAttempts int `json:"invalid_token_attempts"`
	Total                int `json:"total"`
}

type Security struct {
	Status           string `json:"status"`
	CriticalFindings int    `json:"critical_findings"`
	HighFindings     int    `json:"high_findings"`
	Warnings         int    `json:"warnings"`
	Message          string `json:"message"`
}

type Telemetry struct {
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

type CAStatus struct {
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

type Certificate struct {
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

type Enrollment struct {
	ID               string    `json:"id"`
	Hostname         string    `json:"hostname"`
	MachineID        string    `json:"machine_id"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	LastSeenAt       time.Time `json:"last_seen_at,omitempty"`
	CertificateCount int       `json:"certificate_count"`
	FailedAttempts   int       `json:"failed_attempts"`
}

type Token struct {
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

type AuditEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Result    string    `json:"result"`
	Severity  string    `json:"severity"`
}
