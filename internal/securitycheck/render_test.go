package securitycheck

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSON(t *testing.T) {
	report := Report{Summary: Summary{Passed: 1}, Checks: []Result{{ID: "x", Status: StatusPass, Severity: SeverityLow}}}
	var b bytes.Buffer
	if err := RenderJSON(&b, report); err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(b.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Summary.Passed != 1 || decoded.Checks[0].ID != "x" {
		t.Fatal("unexpected json render")
	}
}

func TestRenderMarkdown(t *testing.T) {
	report := Report{Summary: Summary{Warnings: 1}, Checks: []Result{{ID: "ca.root.private_key_absent", Category: "ca", Status: StatusWarn, Severity: SeverityCritical, Message: "msg", Remediation: "fix"}}}
	var b bytes.Buffer
	if err := RenderMarkdown(&b, report); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "# IronRoot Security Check Report") || !strings.Contains(out, "`ca.root.private_key_absent`") {
		t.Fatalf("unexpected markdown: %s", out)
	}
}
