package irtop

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelKeyboardNavigation(t *testing.T) {
	model := NewModel(nil, time.Second, ViewOverview)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	got := updated.(Model)
	if got.view != ViewCertificates {
		t.Fatalf("expected certificates view, got %v", got.view)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	got = updated.(Model)
	if got.view != ViewHelp {
		t.Fatalf("expected help view, got %v", got.view)
	}
}

func TestRenderTextOverview(t *testing.T) {
	out := RenderText(Snapshot{Overview: Overview{
		Server:       ServerStatus{Status: "healthy", Version: "v0.1.0", APIHealth: "ok", Readiness: "ready"},
		Certificates: CertificateSummary{Active: 3, ExpiringSoon: 1},
		Enrollments:  EnrollmentSummary{Successful: 2},
		Security:     Security{Status: "pass"},
		Telemetry:    Telemetry{Enabled: true, TracesEnabled: true, MetricsEnabled: true},
	}})
	for _, want := range []string{"IronRoot", "healthy", "Certificates", "Enrollments", "Telemetry"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q:\n%s", want, out)
		}
	}
}
