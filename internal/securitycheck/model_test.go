package securitycheck

import "testing"

func TestFailsThreshold(t *testing.T) {
	results := []Result{
		{Severity: SeverityMedium, Status: StatusFail},
		{Severity: SeverityHigh, Status: StatusWarn},
	}
	if !FailsThreshold(results, SeverityMedium) {
		t.Fatal("expected medium failure to trip medium threshold")
	}
	if FailsThreshold(results, SeverityHigh) {
		t.Fatal("warn status must not trip fail threshold")
	}
}
