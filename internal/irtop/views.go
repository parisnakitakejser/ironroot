package irtop

import "strings"

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

type viewSpec struct {
	View    View
	Name    string
	Tab     string
	Help    string
	Key     string
	Aliases []string
}

var mainViewSpecs = []viewSpec{
	{View: ViewOverview, Name: "overview", Tab: "1 Overview", Help: "overview", Key: "1"},
	{View: ViewCertificates, Name: "certificates", Tab: "2 Certs", Help: "certificates", Key: "2", Aliases: []string{"certs"}},
	{View: ViewEnrollments, Name: "enrollments", Tab: "3 Enroll", Help: "enrollments", Key: "3"},
	{View: ViewTokens, Name: "tokens", Tab: "4 Tokens", Help: "tokens", Key: "4"},
	{View: ViewCAHealth, Name: "ca", Tab: "5 CA", Help: "CA health", Key: "5", Aliases: []string{"ca-health", "ca_health"}},
	{View: ViewSecurity, Name: "security", Tab: "6 Security", Help: "security", Key: "6"},
	{View: ViewTelemetry, Name: "telemetry", Tab: "7 Telemetry", Help: "telemetry", Key: "7", Aliases: []string{"otel"}},
	{View: ViewAuditLog, Name: "audit", Tab: "8 Audit", Help: "audit log", Key: "8", Aliases: []string{"audit-log", "audit_log"}},
	{View: ViewServer, Name: "server", Tab: "9 Server", Help: "server", Key: "9"},
}

var (
	viewByName = buildViewNameIndex()
	viewByKey  = buildViewKeyIndex()
	nameByView = buildViewDisplayNameIndex()
)

func ParseView(name string) View {
	if view, ok := viewByName[normalizeViewName(name)]; ok {
		return view
	}
	return ViewOverview
}

func validViewName(value string) bool {
	_, ok := viewByName[normalizeViewName(value)]
	return ok
}

func viewForKey(key string) (View, bool) {
	view, ok := viewByKey[key]
	return view, ok
}

func viewName(v View) string {
	if v == ViewHelp {
		return "help"
	}
	if name, ok := nameByView[v]; ok {
		return name
	}
	return "overview"
}

func buildViewNameIndex() map[string]View {
	views := make(map[string]View, len(mainViewSpecs)*3)
	for _, spec := range mainViewSpecs {
		views[spec.Name] = spec.View
		views[spec.Key] = spec.View
		for _, alias := range spec.Aliases {
			views[alias] = spec.View
		}
	}
	return views
}

func buildViewKeyIndex() map[string]View {
	views := make(map[string]View, len(mainViewSpecs))
	for _, spec := range mainViewSpecs {
		views[spec.Key] = spec.View
	}
	return views
}

func buildViewDisplayNameIndex() map[View]string {
	names := make(map[View]string, len(mainViewSpecs))
	for _, spec := range mainViewSpecs {
		names[spec.View] = spec.Name
	}
	return names
}

func normalizeViewName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
