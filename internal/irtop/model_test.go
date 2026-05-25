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

func TestModelProfileSwitching(t *testing.T) {
	model := NewProfileModel(ProfileSet{
		Profiles: []Profile{
			{Name: "local", Config: Config{Server: "http://localhost:8443", Refresh: time.Second, DefaultView: "overview", Output: "tui"}},
			{Name: "prod", Config: Config{Server: "https://ironroot.example.com:8443", Refresh: time.Second, DefaultView: "security", Output: "tui"}},
		},
		Active: 0,
	})
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	got := updated.(Model)
	if got.activeProfile != 1 {
		t.Fatalf("active profile = %d, want 1", got.activeProfile)
	}
	if got.view != ViewSecurity {
		t.Fatalf("view = %v, want security", got.view)
	}
	if got.state != stateLoading {
		t.Fatalf("state = %v, want loading", got.state)
	}
	if cmd == nil {
		t.Fatal("expected refresh command after profile switch")
	}
}

func TestModelProfileSelectorView(t *testing.T) {
	model := NewProfileModel(ProfileSet{
		Profiles: []Profile{
			{Name: "local", Config: Config{Server: "http://localhost:8443", Refresh: time.Second, DefaultView: "overview", Output: "tui"}},
			{Name: "prod", Config: Config{Server: "https://ironroot.example.com:8443", Refresh: time.Second, DefaultView: "overview", Output: "tui"}},
		},
	})
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	got := updated.(Model)
	if !got.selectingProfile {
		t.Fatal("expected profile selector to open")
	}
	out := got.View()
	for _, want := range []string{"Profiles", "local", "prod"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected selector to contain %q:\n%s", want, out)
		}
	}
}

func TestModelStartupStates(t *testing.T) {
	model := NewModel(nil, time.Second, ViewOverview)
	if !strings.Contains(model.View(), "Loading") {
		t.Fatalf("expected loading view:\n%s", model.View())
	}
	updated, _ := model.Update(snapshotMsg{err: errTestStartup})
	got := updated.(Model)
	if got.state != stateError || !strings.Contains(got.View(), "Startup Error") {
		t.Fatalf("expected startup error view:\n%s", got.View())
	}
	updated, _ = model.Update(snapshotMsg{snapshot: Snapshot{}})
	got = updated.(Model)
	if got.state != stateEmpty || !strings.Contains(got.View(), "No Data") {
		t.Fatalf("expected empty view:\n%s", got.View())
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

var errTestStartup = testError("boom")

type testError string

func (e testError) Error() string { return string(e) }
