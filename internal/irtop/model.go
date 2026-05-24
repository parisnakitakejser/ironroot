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
	body := m.renderBody()
	if m.err != nil {
		body += "\n\n" + alertBox(m.contentWidth(), "API error", m.err.Error())
	}
	return m.frame(body)
}

func (m Model) renderBody() string {
	switch m.view {
	case ViewCertificates:
		return m.certificatesView()
	case ViewEnrollments:
		return m.enrollmentsView()
	case ViewTokens:
		return m.tokensView()
	case ViewCAHealth:
		return m.caView()
	case ViewSecurity:
		return m.securityView()
	case ViewTelemetry:
		return m.telemetryView()
	case ViewAuditLog:
		return m.auditView()
	case ViewServer:
		return m.serverView()
	case ViewHelp:
		return m.helpView()
	default:
		return m.overviewView()
	}
}

func (m Model) frame(content string) string {
	if m.width <= 0 || m.height <= 0 {
		return content
	}
	innerWidth := max(40, m.width-4)
	bodyHeight := max(1, m.height-5)
	header := m.header(innerWidth)
	tabs := m.tabs(innerWidth)
	body := appStyle.Width(innerWidth).Height(bodyHeight).MaxWidth(innerWidth).Render(content)
	footer := footerStyle.Width(innerWidth).Render(" q quit  ? help  r refresh  1-9 views  / filter  s sort  enter details")
	screen := strings.Join([]string{header, tabs, body, footer}, "\n")
	return appStyle.Width(m.width).Height(m.height).Padding(0, 2).Render(screen)
}

func (m Model) header(width int) string {
	o := m.snapshot.Overview
	status := empty(o.Server.Status, "connecting")
	version := empty(o.Server.Version, "dev")
	updated := "not refreshed"
	if !m.snapshot.UpdatedAt.IsZero() {
		updated = m.snapshot.UpdatedAt.Format("15:04:05")
	}
	left := " IRONROOT TOP "
	mid := fmt.Sprintf(" %s  version %s  api %s/%s ", statusBadge(status), version, empty(o.Server.APIHealth, "unknown"), empty(o.Server.Readiness, "unknown"))
	right := subtleStyle.Render("updated " + updated)
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(mid)-lipgloss.Width(right))
	return headerStyle.Width(width).Render(titleStyle.Render(left) + mid + strings.Repeat(" ", gap) + right)
}

func (m Model) tabs(width int) string {
	names := []string{"1 Overview", "2 Certs", "3 Enroll", "4 Tokens", "5 CA", "6 Security", "7 Telemetry", "8 Audit", "9 Server"}
	for i := range names {
		if View(i) == m.view {
			names[i] = activeTabStyle.Render(names[i])
		} else {
			names[i] = tabStyle.Render(names[i])
		}
	}
	return tabsStyle.Width(width).Render(strings.Join(names, " "))
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 80
	}
	return max(40, m.width-4)
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
		return panel(m.contentWidth(), "Waiting", "Waiting for IronRoot status...")
	}
	contentWidth := m.contentWidth()
	panelWidth := max(34, (contentWidth-2)/2)
	wideWidth := max(34, contentWidth)
	return strings.Join([]string{
		panel(wideWidth, "IronRoot", strings.Join([]string{
			metricLine("Server", statusBadge(o.Server.Status), "Version", empty(o.Server.Version, "dev")),
			metricLine("API", o.Server.APIHealth+"/"+o.Server.Readiness, "CA", empty(o.CA.ChainStatus, "unknown")),
		}, "\n")),
		row(
			panel(panelWidth, "Certificates", strings.Join([]string{
				metricLine("Active", fmt.Sprint(o.Certificates.Active), "Expiring", warnNumber(o.Certificates.ExpiringSoon)),
				metricLine("Issued today", fmt.Sprint(o.Certificates.IssuedToday), "Revoked", warnNumber(o.Certificates.Revoked)),
			}, "\n")),
			panel(panelWidth, "Enrollments", strings.Join([]string{
				metricLine("Success", fmt.Sprint(o.Enrollments.Successful), "Failed", warnNumber(o.Enrollments.Failed)),
				metricLine("Pending", fmt.Sprint(o.Enrollments.Pending), "Invalid tokens", warnNumber(o.Enrollments.InvalidTokenAttempts)),
			}, "\n")),
		),
		row(
			panel(panelWidth, "Security", strings.Join([]string{
				metricLine("Status", statusBadge(o.Security.Status), "Critical", warnNumber(o.Security.CriticalFindings)),
				metricLine("High", warnNumber(o.Security.HighFindings), "Warnings", warnNumber(o.Security.Warnings)),
			}, "\n")),
			panel(panelWidth, "Telemetry", strings.Join([]string{
				metricLine("Enabled", boolBadge(o.Telemetry.Enabled), "Exporter", empty(o.Telemetry.ExporterStatus, "unknown")),
				metricLine("Traces", boolBadge(o.Telemetry.TracesEnabled), "Metrics", boolBadge(o.Telemetry.MetricsEnabled)),
			}, "\n")),
		),
	}, "\n\n")
}

func (m Model) certificatesView() string {
	lines := []string{tableHeader("SERIAL          DNS                    STATUS   EXPIRES      DAYS  ISSUER")}
	for _, cert := range m.snapshot.Certificates {
		lines = append(lines, tableRow(fmt.Sprintf("%-15s %-22s %-8s %-12s %4d  %.12s", short(cert.Serial), truncate(strings.Join(cert.DNSNames, ","), 22), cert.Status, date(cert.ExpiresAt), cert.DaysRemaining, cert.Issuer)))
	}
	return panel(m.contentWidth(), "Certificates", strings.Join(lines, "\n"))
}

func (m Model) enrollmentsView() string {
	lines := []string{tableHeader("ENROLLMENT ID                         HOST                 STATUS   CERTS  CREATED")}
	for _, en := range m.snapshot.Enrollments {
		lines = append(lines, tableRow(fmt.Sprintf("%-37s %-20s %-8s %5d  %s", en.ID, truncate(en.Hostname, 20), en.Status, en.CertificateCount, date(en.CreatedAt))))
	}
	return panel(m.contentWidth(), "Enrollments", strings.Join(lines, "\n"))
}

func (m Model) tokensView() string {
	lines := []string{tableHeader("TOKEN ID                              HOST                 STATUS   USED  EXPIRES")}
	for _, token := range m.snapshot.Tokens {
		lines = append(lines, tableRow(fmt.Sprintf("%-37s %-20s %-8s %-5t %s", token.ID, truncate(token.Host, 20), token.Status, token.Used, date(token.ExpiresAt))))
	}
	return panel(m.contentWidth(), "Bootstrap Tokens", strings.Join(lines, "\n"))
}

func (m Model) caView() string {
	ca := m.snapshot.CA
	if ca.ActiveIssuer == "" {
		ca = m.snapshot.Overview.CA
	}
	lines := []string{
		metricLine("Root fingerprint", short(ca.RootFingerprint), "Root expires", date(ca.RootExpiresAt)),
		metricLine("Intermediate", short(ca.IntermediateFingerprint), "Intermediate expires", date(ca.IntermediateExpiresAt)),
		metricLine("Active issuer", short(ca.ActiveIssuer), "Chain", statusBadge(ca.ChainStatus)),
		metricLine("Disabled issuers", fmt.Sprint(ca.DisabledIssuers), "Retired issuers", fmt.Sprint(ca.RetiredIssuers)),
	}
	for _, warning := range ca.Warnings {
		lines = append(lines, warnStyle.Render("warning: "+warning))
	}
	return panel(m.contentWidth(), "CA Health", strings.Join(lines, "\n"))
}

func (m Model) securityView() string {
	s := m.snapshot.Security
	if s.Status == "" {
		s = m.snapshot.Overview.Security
	}
	return panel(m.contentWidth(), "Security", strings.Join([]string{
		metricLine("Status", statusBadge(s.Status), "Critical", warnNumber(s.CriticalFindings)),
		metricLine("High", warnNumber(s.HighFindings), "Warnings", warnNumber(s.Warnings)),
		"",
		s.Message,
	}, "\n"))
}

func (m Model) telemetryView() string {
	t := m.snapshot.Telemetry
	if t.ExporterStatus == "" {
		t = m.snapshot.Overview.Telemetry
	}
	return panel(m.contentWidth(), "Telemetry", strings.Join([]string{
		metricLine("Enabled", boolBadge(t.Enabled), "Exporter", empty(t.ExporterStatus, "unknown")),
		metricLine("Traces", boolBadge(t.TracesEnabled), "Metrics", boolBadge(t.MetricsEnabled)),
		metricLine("Logs", boolBadge(t.LogsEnabled), "Prometheus", boolBadge(t.PrometheusEnabled)),
		metricLine("OTLP", empty(t.OTLPEndpoint, "not configured"), "Protocol", empty(t.OTLPProtocol, "n/a")),
		metricLine("Metrics path", empty(t.PrometheusPath, "n/a"), "", ""),
	}, "\n"))
}

func (m Model) auditView() string {
	lines := []string{tableHeader("TIME                  SEVERITY  ACTION                    ACTOR      TARGET")}
	for _, event := range m.snapshot.Audit {
		lines = append(lines, tableRow(fmt.Sprintf("%-21s %-9s %-25s %-10s %s", event.Timestamp.Format("2006-01-02 15:04:05"), event.Severity, truncate(event.Action, 25), truncate(event.Actor, 10), event.Target)))
	}
	return panel(m.contentWidth(), "Recent Audit Events", strings.Join(lines, "\n"))
}

func (m Model) serverView() string {
	s := m.snapshot.Server
	if s.Status == "" {
		s = m.snapshot.Overview.Server
	}
	return panel(m.contentWidth(), "Server", strings.Join([]string{
		metricLine("Status", statusBadge(s.Status), "Version", empty(s.Version, "dev")),
		metricLine("API", s.APIHealth+" / "+s.Readiness, "Address", empty(s.APIAddress, "unknown")),
		metricLine("TLS", boolBadge(s.TLSEnabled), "Database", empty(s.DatabaseType, "unknown")),
		metricLine("Go", empty(s.GoVersion, "unknown"), "Goroutines", fmt.Sprint(s.Goroutines)),
	}, "\n"))
}

func (m Model) helpView() string {
	return panel(m.contentWidth(), "Help", strings.Join([]string{
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
	}, "\n"))
}

var (
	appBg          = lipgloss.Color("235")
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("24")).Bold(true)
	appStyle       = lipgloss.NewStyle().Background(appBg)
	tabsStyle      = lipgloss.NewStyle().Background(appBg)
	tabStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Background(lipgloss.Color("236")).Padding(0, 1)
	activeTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("81")).Bold(true).Padding(0, 1)
	panelStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Background(appBg).Padding(0, 1)
	tableHeadStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	tableRowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	subtleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	warnStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	okStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	badStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	footerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Background(lipgloss.Color("236"))
)

func panel(width int, title, body string) string {
	return panelStyle.Width(panelContentWidth(width)).Render(titleStyle.Render(" "+title+" ") + "\n" + body)
}

func alertBox(width int, title, body string) string {
	return panelStyle.BorderForeground(lipgloss.Color("203")).Width(panelContentWidth(width)).Render(badStyle.Render(" "+title+" ") + "\n" + body)
}

func row(left, right string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func metricLine(leftLabel, leftValue, rightLabel, rightValue string) string {
	left := subtleStyle.Render(leftLabel+": ") + leftValue
	if rightLabel == "" {
		return left
	}
	return fmt.Sprintf("%-42s %s", left, subtleStyle.Render(rightLabel+": ")+rightValue)
}

func tableHeader(s string) string {
	return tableHeadStyle.Render(s)
}

func tableRow(s string) string {
	return tableRowStyle.Render(s)
}

func statusBadge(s string) string {
	switch strings.ToLower(s) {
	case "healthy", "ready", "valid", "pass", "active", "ok":
		return okStyle.Render(s)
	case "critical", "high", "failed", "error", "unconfigured", "revoked", "expired":
		return badStyle.Render(empty(s, "unknown"))
	case "":
		return warnStyle.Render("unknown")
	default:
		return warnStyle.Render(s)
	}
}

func boolBadge(v bool) string {
	if v {
		return okStyle.Render("on")
	}
	return subtleStyle.Render("off")
}

func warnNumber(v int) string {
	if v > 0 {
		return warnStyle.Render(fmt.Sprint(v))
	}
	return okStyle.Render("0")
}

func short(s string) string {
	if len(s) <= 14 {
		return s
	}
	return s[:14]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func panelContentWidth(outerWidth int) int {
	// lipgloss Width applies to content. The rounded border plus horizontal
	// padding add four cells, so callers pass the desired visible panel width.
	return max(16, outerWidth-4)
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
