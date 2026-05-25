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

type snapshotMsg struct {
	profileIndex int
	snapshot     Snapshot
	err          error
}

type tickMsg time.Time

type loadState int

const (
	stateLoading loadState = iota
	stateLoaded
	stateEmpty
	stateError
)

type Model struct {
	client           *Client
	profiles         []Profile
	activeProfile    int
	profileCursor    int
	selectingProfile bool
	refresh          time.Duration
	view             View
	snapshot         Snapshot
	err              error
	state            loadState
	width            int
	height           int
}

func NewModel(client *Client, refresh time.Duration, initial View) Model {
	if refresh <= 0 {
		refresh = defaultRefresh
	}
	return Model{client: client, refresh: refresh, view: initial, state: stateLoading}
}

func NewProfileModel(profiles ProfileSet) Model {
	if profiles.Len() == 0 {
		m := NewModel(nil, defaultRefresh, ViewOverview)
		m.err = fmt.Errorf("no irtop profiles configured")
		m.state = stateError
		return m
	}
	active := profiles.ActiveIndex()
	cfg := profiles.Profiles[active].Config
	client, err := NewClientChecked(cfg)
	m := NewModel(client, cfg.Refresh, ParseView(cfg.DefaultView))
	m.profiles = profiles.Profiles
	m.activeProfile = active
	m.profileCursor = active
	if err != nil {
		m.err = err
		m.state = stateError
	}
	return m
}

func RenderText(s Snapshot) string {
	m := NewModel(nil, defaultRefresh, ViewOverview)
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
		if m.selectingProfile {
			return m.updateProfileSelector(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "p":
			if len(m.profiles) > 1 {
				m.selectingProfile = true
				m.profileCursor = m.activeProfile
			}
		case "[":
			if len(m.profiles) > 1 {
				return m.activateProfile((m.activeProfile - 1 + len(m.profiles)) % len(m.profiles))
			}
		case "]":
			if len(m.profiles) > 1 {
				return m.activateProfile((m.activeProfile + 1) % len(m.profiles))
			}
		case "?", "esc":
			if m.view == ViewHelp {
				m.view = ViewOverview
			} else {
				m.view = ViewHelp
			}
		case "r":
			return m, m.refreshNow()
		default:
			if view, ok := viewForKey(msg.String()); ok {
				m.view = view
			}
		}
	case tickMsg:
		return m, tea.Batch(m.refreshNow(), tick(m.refresh))
	case snapshotMsg:
		if msg.profileIndex != m.activeProfile {
			return m, nil
		}
		m.err = msg.err
		if msg.err == nil {
			m.snapshot = msg.snapshot
			if snapshotEmpty(msg.snapshot) {
				m.state = stateEmpty
			} else {
				m.state = stateLoaded
			}
		} else if m.snapshot.UpdatedAt.IsZero() {
			m.state = stateError
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.selectingProfile {
		return m.frame(m.profileSelectorView())
	}
	body := m.renderBody()
	if m.err != nil && m.state != stateError {
		body += "\n\n" + alertBox(m.contentWidth(), "API error", m.err.Error())
	}
	return m.frame(body)
}

func (m Model) renderBody() string {
	switch m.state {
	case stateLoading:
		return m.loadingView()
	case stateError:
		return m.startupErrorView()
	case stateEmpty:
		return m.emptyView()
	}
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
	footerText := " q quit  ? help  r refresh  1-9 views"
	if len(m.profiles) > 1 {
		footerText += "  p profiles  [ ] switch"
	}
	footer := footerStyle.Width(innerWidth).Render(footerText)
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
	profile := m.activeProfileName()
	left := " IRONROOT TOP "
	mid := fmt.Sprintf(" %s  profile %s  version %s  api %s/%s ", statusBadge(status), titleStyle.Render(profile), version, empty(o.Server.APIHealth, "unknown"), empty(o.Server.Readiness, "unknown"))
	right := subtleStyle.Render("updated " + updated)
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(mid)-lipgloss.Width(right))
	return headerStyle.Width(width).Render(titleStyle.Render(left) + mid + strings.Repeat(" ", gap) + right)
}

func (m Model) tabs(width int) string {
	names := make([]string, 0, len(mainViewSpecs))
	for _, spec := range mainViewSpecs {
		label := spec.Tab
		if spec.View == m.view {
			label = activeTabStyle.Render(label)
		} else {
			label = tabStyle.Render(label)
		}
		names = append(names, label)
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
	client := m.client
	profileIndex := m.activeProfile
	view := m.view
	return func() tea.Msg {
		if client == nil {
			return snapshotMsg{profileIndex: profileIndex, err: fmt.Errorf("active profile %q is not usable", m.activeProfileName())}
		}
		ctx, span := otel.Tracer("github.com/parisnakitakejser/ironroot/irtop").Start(context.Background(), "irtop.refresh")
		start := time.Now()
		snapshot, err := client.Snapshot(ctx)
		span.SetAttributes(attribute.String("view", viewName(view)), attribute.String("profile", m.activeProfileName()), attribute.Float64("duration_ms", float64(time.Since(start).Milliseconds())))
		if err != nil {
			span.RecordError(err)
		}
		span.End()
		return snapshotMsg{profileIndex: profileIndex, snapshot: snapshot, err: err}
	}
}

func (m Model) updateProfileSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "p":
		m.selectingProfile = false
	case "up", "k":
		if len(m.profiles) > 0 {
			m.profileCursor = (m.profileCursor - 1 + len(m.profiles)) % len(m.profiles)
		}
	case "down", "j":
		if len(m.profiles) > 0 {
			m.profileCursor = (m.profileCursor + 1) % len(m.profiles)
		}
	case "enter":
		m.selectingProfile = false
		return m.activateProfile(m.profileCursor)
	}
	return m, nil
}

func (m Model) activateProfile(index int) (tea.Model, tea.Cmd) {
	if index < 0 || index >= len(m.profiles) || index == m.activeProfile {
		return m, nil
	}
	cfg := m.profiles[index].Config
	client, err := NewClientChecked(cfg)
	m.client = client
	m.activeProfile = index
	m.profileCursor = index
	m.refresh = cfg.Refresh
	m.view = ParseView(cfg.DefaultView)
	m.snapshot = Snapshot{}
	m.err = err
	if err != nil {
		m.state = stateError
		return m, nil
	}
	m.state = stateLoading
	return m, m.refreshNow()
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) loadingView() string {
	lines := []string{
		"Loading IronRoot status...",
		"",
		"Profile: " + m.activeProfileName(),
		"Server:  " + empty(m.activeConfig().Server, "not configured"),
	}
	return panel(m.contentWidth(), "Loading", strings.Join(lines, "\n"))
}

func (m Model) startupErrorView() string {
	body := "Startup failed before irtop could display data."
	if m.err != nil {
		body += "\n\n" + m.err.Error()
	}
	body += "\n\nPress r to retry, p to switch profiles, or q to quit."
	return alertBox(m.contentWidth(), "Startup Error", body)
}

func (m Model) emptyView() string {
	return panel(m.contentWidth(), "No Data", strings.Join([]string{
		"IronRoot returned an empty status snapshot.",
		"",
		"Press r to refresh or p to switch profiles.",
	}, "\n"))
}

func (m Model) profileSelectorView() string {
	if len(m.profiles) == 0 {
		return alertBox(m.contentWidth(), "Profiles", "No profiles are configured.")
	}
	lines := []string{"Use up/down or k/j to choose a profile. Press enter to switch."}
	lines = append(lines, "")
	for i, profile := range m.profiles {
		cursor := "  "
		if i == m.profileCursor {
			cursor = "> "
		}
		active := ""
		if i == m.activeProfile {
			active = " active"
		}
		line := fmt.Sprintf("%s%-18s %s%s", cursor, profile.Name, profile.Config.Server, active)
		if i == m.profileCursor {
			line = activeTabStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return panel(m.contentWidth(), "Profiles", strings.Join(lines, "\n"))
}

func (m Model) activeProfileName() string {
	if len(m.profiles) == 0 || m.activeProfile < 0 || m.activeProfile >= len(m.profiles) {
		return "default"
	}
	return m.profiles[m.activeProfile].Name
}

func (m Model) activeConfig() Config {
	if len(m.profiles) == 0 || m.activeProfile < 0 || m.activeProfile >= len(m.profiles) {
		return Config{}
	}
	return m.profiles[m.activeProfile].Config
}

func snapshotEmpty(snapshot Snapshot) bool {
	return snapshot.Overview.Server.Status == "" &&
		snapshot.Server.Status == "" &&
		len(snapshot.Certificates) == 0 &&
		len(snapshot.Enrollments) == 0 &&
		len(snapshot.Tokens) == 0 &&
		snapshot.CA.ChainStatus == "" &&
		len(snapshot.CAHierarchy.Roots) == 0 &&
		snapshot.Security.Status == "" &&
		snapshot.Telemetry.ExporterStatus == "" &&
		len(snapshot.Audit) == 0
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
	if len(m.snapshot.CAHierarchy.Roots) > 0 {
		lines = append(lines, "")
		lines = append(lines, m.caHierarchyTree()...)
	} else {
		for _, warning := range m.snapshot.CAHierarchy.Warnings {
			lines = append(lines, warnStyle.Render("hierarchy: "+warning))
		}
	}
	return panel(m.contentWidth(), "CA Health", strings.Join(lines, "\n"))
}

func (m Model) caHierarchyTree() []string {
	h := m.snapshot.CAHierarchy
	lines := []string{
		tableHeader("Trust hierarchy"),
		metricLine("Roots", fmt.Sprint(h.Summary.RootCAs), "Intermediates", fmt.Sprint(h.Summary.IntermediateCAs)),
		metricLine("Policies", fmt.Sprint(h.Summary.TokenPolicies), "RBAC roles", fmt.Sprint(h.Summary.Roles)),
	}
	if h.LegacyFallback {
		lines = append(lines, warnStyle.Render("legacy ca_config projection"))
	}
	for _, root := range h.Roots {
		lines = append(lines, fmt.Sprintf("%s %s %s", statusBadge(root.Status), titleStyle.Render(root.Name), subtleStyle.Render("["+empty(root.Environment, "unknown")+"]")))
		lines = append(lines, subtleStyle.Render("   root "+short(root.Fingerprint)+"  expires "+date(root.NotAfter)+"  trust "+empty(root.TrustDomain, "n/a")))
		for i, intermediate := range root.Intermediates {
			branch := "|-"
			if i == len(root.Intermediates)-1 {
				branch = "`-"
			}
			owner := empty(intermediate.Owner, "unowned")
			if intermediate.Namespace != "" {
				owner += "/" + intermediate.Namespace
			}
			lines = append(lines, fmt.Sprintf("   %s %s %s %s", branch, statusBadge(intermediate.Status), intermediate.Name, subtleStyle.Render(owner)))
			lines = append(lines, fmt.Sprintf("      certs active=%d revoked=%d ttl=%s renew=%s approval=%s", intermediate.ActiveCerts, intermediate.RevokedCerts, empty(intermediate.MaxTTL, "n/a"), boolWord(intermediate.RenewalAllowed), boolWord(intermediate.RequireApproval)))
			if len(intermediate.AllowedDNS) > 0 {
				lines = append(lines, subtleStyle.Render("      dns "+truncate(strings.Join(intermediate.AllowedDNS, ", "), 72)))
			}
			if len(intermediate.TokenPolicies) > 0 || len(intermediate.Roles) > 0 {
				lines = append(lines, subtleStyle.Render(fmt.Sprintf("      access policies=%d roles=%d", len(intermediate.TokenPolicies), len(intermediate.Roles))))
			}
		}
	}
	for _, warning := range h.Warnings {
		lines = append(lines, warnStyle.Render("warning: "+warning))
	}
	return lines
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
	lines := []string{
		"q       quit",
		"?       help",
		"r       refresh now",
	}
	for _, spec := range mainViewSpecs {
		lines = append(lines, fmt.Sprintf("%s       %s", spec.Key, spec.Help))
	}
	lines = append(lines,
		"p       profile selector",
		"[ / ]   previous / next profile",
		"/       search/filter (planned)",
		"s       sort (planned)",
		"enter   details (planned)",
		"esc     back",
	)
	return panel(m.contentWidth(), "Help", strings.Join(lines, "\n"))
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

func boolWord(v bool) string {
	if v {
		return "yes"
	}
	return "no"
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
