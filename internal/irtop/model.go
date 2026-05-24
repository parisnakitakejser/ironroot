package irtop

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type View int

const (
	ViewOverview View = iota
	ViewCertificates
	ViewEnrollments
	ViewTokens
	ViewCAHealth
	ViewSecurity
	ViewTelemetry
	ViewAuditLog
	ViewServer
	ViewHelp
)

type snapshotMsg struct {
	snapshot Snapshot
	err      error
}

type tickMsg time.Time

type Model struct {
	client   *Client
	refresh  time.Duration
	view     View
	snapshot Snapshot
	err      error
	width    int
	height   int
}

func NewModel(client *Client, refresh time.Duration, initial View) Model {
	if refresh <= 0 {
		refresh = 5 * time.Second
	}
	return Model{client: client, refresh: refresh, view: initial}
}

func ParseView(name string) View {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "certificates", "certs":
		return ViewCertificates
	case "enrollments":
		return ViewEnrollments
	case "tokens":
		return ViewTokens
	case "ca", "ca-health", "ca_health":
		return ViewCAHealth
	case "security":
		return ViewSecurity
	case "telemetry", "otel":
		return ViewTelemetry
	case "audit", "audit-log", "audit_log":
		return ViewAuditLog
	case "server":
		return ViewServer
	default:
		return ViewOverview
	}
}

func RenderText(s Snapshot) string {
	m := NewModel(nil, 5*time.Second, ViewOverview)
	m.snapshot = s
	return m.overviewView()
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refreshNow(), tick(m.refresh))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?", "esc":
			if m.view == ViewHelp {
				m.view = ViewOverview
			} else {
				m.view = ViewHelp
			}
		case "r":
			return m, m.refreshNow()
		case "1":
			m.view = ViewOverview
		case "2":
			m.view = ViewCertificates
		case "3":
			m.view = ViewEnrollments
		case "4":
			m.view = ViewTokens
		case "5":
			m.view = ViewCAHealth
		case "6":
			m.view = ViewSecurity
		case "7":
			m.view = ViewTelemetry
		case "8":
			m.view = ViewAuditLog
		case "9":
			m.view = ViewServer
		}
	case tickMsg:
		return m, tea.Batch(m.refreshNow(), tick(m.refresh))
	case snapshotMsg:
		m.err = msg.err
		if msg.err == nil {
			m.snapshot = msg.snapshot
		}
	}
	return m, nil
}

func (m Model) View() string {
	var body string
	switch m.view {
	case ViewCertificates:
		body = m.certificatesView()
	case ViewEnrollments:
		body = m.enrollmentsView()
	case ViewTokens:
		body = m.tokensView()
	case ViewCAHealth:
		body = m.caView()
	case ViewSecurity:
		body = m.securityView()
	case ViewTelemetry:
		body = m.telemetryView()
	case ViewAuditLog:
		body = m.auditView()
	case ViewServer:
		body = m.serverView()
	case ViewHelp:
		body = helpView()
	default:
		body = m.overviewView()
	}
	if m.err != nil {
		body += "\n\n" + warnStyle.Render("API error: "+m.err.Error())
	}
	return titleStyle.Render("irtop - IronRoot Top") + "\n" + navLine(m.view) + "\n\n" + body + "\n\n" + subtleStyle.Render("q quit | ? help | r refresh | 1-9 views")
}

func (m Model) refreshNow() tea.Cmd {
	return func() tea.Msg {
		ctx, span := otel.Tracer("github.com/parisnakitakejser/ironroot/irtop").Start(context.Background(), "irtop.refresh")
		start := time.Now()
		snapshot, err := m.client.Snapshot(ctx)
		span.SetAttributes(attribute.String("view", viewName(m.view)), attribute.Float64("duration_ms", float64(time.Since(start).Milliseconds())))
		if err != nil {
			span.RecordError(err)
		}
		span.End()
		return snapshotMsg{snapshot: snapshot, err: err}
	}
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) overviewView() string {
	o := m.snapshot.Overview
	if o.Server.Status == "" {
		return "Waiting for IronRoot status..."
	}
	return strings.Join([]string{
		box("IronRoot", fmt.Sprintf("Server: %s    Version: %s\nUptime: n/a    API: %s/%s", o.Server.Status, empty(o.Server.Version, "dev"), o.Server.APIHealth, o.Server.Readiness)),
		row(
			box("Certificates", fmt.Sprintf("Active: %d\nExpiring: %d\nRevoked: %d", o.Certificates.Active, o.Certificates.ExpiringSoon, o.Certificates.Revoked)),
			box("Enrollments", fmt.Sprintf("Success: %d\nFailed: %d\nInvalid tokens: %d", o.Enrollments.Successful, o.Enrollments.Failed, o.Enrollments.InvalidTokenAttempts)),
		),
		row(
			box("Security", fmt.Sprintf("Status: %s\nCritical: %d\nHigh: %d\nWarnings: %d", o.Security.Status, o.Security.CriticalFindings, o.Security.HighFindings, o.Security.Warnings)),
			box("Telemetry", fmt.Sprintf("Enabled: %t\nTraces: %t\nMetrics: %t", o.Telemetry.Enabled, o.Telemetry.TracesEnabled, o.Telemetry.MetricsEnabled)),
		),
	}, "\n")
}

func (m Model) certificatesView() string {
	lines := []string{"SERIAL          DNS                    STATUS   EXPIRES      DAYS  ISSUER"}
	for _, cert := range m.snapshot.Certificates {
		lines = append(lines, fmt.Sprintf("%-15s %-22s %-8s %-12s %4d  %.12s", short(cert.Serial), strings.Join(cert.DNSNames, ","), cert.Status, date(cert.ExpiresAt), cert.DaysRemaining, cert.Issuer))
	}
	return strings.Join(lines, "\n")
}

func (m Model) enrollmentsView() string {
	lines := []string{"ENROLLMENT ID                         HOST                 STATUS   CERTS  CREATED"}
	for _, en := range m.snapshot.Enrollments {
		lines = append(lines, fmt.Sprintf("%-37s %-20s %-8s %5d  %s", en.ID, en.Hostname, en.Status, en.CertificateCount, date(en.CreatedAt)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) tokensView() string {
	lines := []string{"TOKEN ID                              HOST                 STATUS   USED  EXPIRES"}
	for _, token := range m.snapshot.Tokens {
		lines = append(lines, fmt.Sprintf("%-37s %-20s %-8s %-5t %s", token.ID, token.Host, token.Status, token.Used, date(token.ExpiresAt)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) caView() string {
	ca := m.snapshot.CA
	if ca.ActiveIssuer == "" {
		ca = m.snapshot.Overview.CA
	}
	lines := []string{
		"Root fingerprint:         " + short(ca.RootFingerprint),
		"Root expires:             " + date(ca.RootExpiresAt),
		"Intermediate fingerprint: " + short(ca.IntermediateFingerprint),
		"Intermediate expires:     " + date(ca.IntermediateExpiresAt),
		"Active issuer:            " + short(ca.ActiveIssuer),
		"Chain status:             " + ca.ChainStatus,
	}
	for _, warning := range ca.Warnings {
		lines = append(lines, warnStyle.Render("warning: "+warning))
	}
	return strings.Join(lines, "\n")
}

func (m Model) securityView() string {
	s := m.snapshot.Security
	if s.Status == "" {
		s = m.snapshot.Overview.Security
	}
	return fmt.Sprintf("Status: %s\nCritical findings: %d\nHigh findings: %d\nWarnings: %d\n\n%s", s.Status, s.CriticalFindings, s.HighFindings, s.Warnings, s.Message)
}

func (m Model) telemetryView() string {
	t := m.snapshot.Telemetry
	if t.ExporterStatus == "" {
		t = m.snapshot.Overview.Telemetry
	}
	return fmt.Sprintf("Enabled: %t\nTraces: %t\nMetrics: %t\nLogs: %t\nOTLP: %s (%s)\nPrometheus: %t %s\nExporter: %s", t.Enabled, t.TracesEnabled, t.MetricsEnabled, t.LogsEnabled, t.OTLPEndpoint, t.OTLPProtocol, t.PrometheusEnabled, t.PrometheusPath, t.ExporterStatus)
}

func (m Model) auditView() string {
	lines := []string{"TIME                  SEVERITY  ACTION                    ACTOR      TARGET"}
	for _, event := range m.snapshot.Audit {
		lines = append(lines, fmt.Sprintf("%-21s %-9s %-25s %-10s %s", event.Timestamp.Format("2006-01-02 15:04:05"), event.Severity, event.Action, event.Actor, event.Target))
	}
	return strings.Join(lines, "\n")
}

func (m Model) serverView() string {
	s := m.snapshot.Server
	if s.Status == "" {
		s = m.snapshot.Overview.Server
	}
	return fmt.Sprintf("Status: %s\nVersion: %s\nAPI: %s / %s\nAddress: %s\nTLS: %t\nDatabase: %s\nGo: %s\nGoroutines: %d", s.Status, empty(s.Version, "dev"), s.APIHealth, s.Readiness, s.APIAddress, s.TLSEnabled, s.DatabaseType, s.GoVersion, s.Goroutines)
}

func helpView() string {
	return strings.Join([]string{
		"q       quit",
		"?       help",
		"r       refresh now",
		"1       overview",
		"2       certificates",
		"3       enrollments",
		"4       tokens",
		"5       CA health",
		"6       security",
		"7       telemetry",
		"8       audit log",
		"9       server",
		"/       search/filter (planned)",
		"s       sort (planned)",
		"enter   details (planned)",
		"esc     back",
	}, "\n")
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(38)
)

func navLine(active View) string {
	names := []string{"1 Overview", "2 Certificates", "3 Enrollments", "4 Tokens", "5 CA", "6 Security", "7 Telemetry", "8 Audit", "9 Server"}
	for i := range names {
		if View(i) == active {
			names[i] = titleStyle.Render(names[i])
		}
	}
	return strings.Join(names, "  ")
}

func box(title, body string) string {
	return boxStyle.Render(titleStyle.Render(title) + "\n" + body)
}

func row(left, right string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func short(s string) string {
	if len(s) <= 14 {
		return s
	}
	return s[:14]
}

func date(t time.Time) string {
	if t.IsZero() {
		return "n/a"
	}
	return t.Format("2006-01-02")
}

func empty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func viewName(v View) string {
	switch v {
	case ViewCertificates:
		return "certificates"
	case ViewEnrollments:
		return "enrollments"
	case ViewTokens:
		return "tokens"
	case ViewCAHealth:
		return "ca"
	case ViewSecurity:
		return "security"
	case ViewTelemetry:
		return "telemetry"
	case ViewAuditLog:
		return "audit"
	case ViewServer:
		return "server"
	case ViewHelp:
		return "help"
	default:
		return "overview"
	}
}
