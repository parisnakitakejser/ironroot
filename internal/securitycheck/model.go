package securitycheck

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/ironroot/ironroot/internal/config"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

type Result struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Category         string   `json:"category"`
	Severity         Severity `json:"severity"`
	Status           Status   `json:"status"`
	Message          string   `json:"message"`
	Remediation      string   `json:"remediation"`
	DocumentationURL string   `json:"documentation_url"`
}

type Summary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Failed   int `json:"failed"`
	Skipped  int `json:"skipped"`
}

type Report struct {
	Summary Summary  `json:"summary"`
	Checks  []Result `json:"checks"`
}

type Target struct {
	Config     config.Config
	ConfigPath string
	Now        time.Time
}

type Check interface {
	ID() string
	Category() string
	Run(context.Context, Target) Result
}

func Summarize(results []Result) Summary {
	var s Summary
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			s.Passed++
		case StatusWarn:
			s.Warnings++
		case StatusFail:
			s.Failed++
		case StatusSkip:
			s.Skipped++
		}
	}
	return s
}

func SeverityRank(s Severity) int {
	switch s {
	case SeverityInfo:
		return 0
	case SeverityLow:
		return 1
	case SeverityMedium:
		return 2
	case SeverityHigh:
		return 3
	case SeverityCritical:
		return 4
	default:
		return 0
	}
}

func ParseSeverity(value string) (Severity, error) {
	switch Severity(strings.ToLower(strings.TrimSpace(value))) {
	case SeverityInfo:
		return SeverityInfo, nil
	case SeverityLow:
		return SeverityLow, nil
	case SeverityMedium:
		return SeverityMedium, nil
	case SeverityHigh:
		return SeverityHigh, nil
	case SeverityCritical:
		return SeverityCritical, nil
	default:
		return "", errors.New("severity must be one of info, low, medium, high, critical")
	}
}

func FailsThreshold(results []Result, threshold Severity) bool {
	for _, r := range results {
		if r.Status == StatusFail && SeverityRank(r.Severity) >= SeverityRank(threshold) {
			return true
		}
	}
	return false
}

func SortResults(results []Result) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Category == results[j].Category {
			return results[i].ID < results[j].ID
		}
		return results[i].Category < results[j].Category
	})
}

func baseResult(id, title, category string, severity Severity) Result {
	return Result{
		ID:               id,
		Title:            title,
		Category:         category,
		Severity:         severity,
		Status:           StatusPass,
		DocumentationURL: "https://example.com/docs/security/" + strings.ReplaceAll(category, "_", "-"),
	}
}
