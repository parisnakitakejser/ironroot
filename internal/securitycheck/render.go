package securitycheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type OutputFormat string

const (
	OutputTable    OutputFormat = "table"
	OutputJSON     OutputFormat = "json"
	OutputMarkdown OutputFormat = "markdown"
)

func ParseOutputFormat(value string) (OutputFormat, error) {
	switch OutputFormat(strings.ToLower(strings.TrimSpace(value))) {
	case OutputTable:
		return OutputTable, nil
	case OutputJSON:
		return OutputJSON, nil
	case OutputMarkdown:
		return OutputMarkdown, nil
	default:
		return "", fmt.Errorf("output must be one of table, json, markdown")
	}
}

func Render(w io.Writer, report Report, format OutputFormat) error {
	switch format {
	case OutputTable:
		return RenderTable(w, report)
	case OutputJSON:
		return RenderJSON(w, report)
	case OutputMarkdown:
		return RenderMarkdown(w, report)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func RenderJSON(w io.Writer, report Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func RenderTable(w io.Writer, report Report) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "STATUS\tSEVERITY\tCATEGORY\tID\tMESSAGE\n")
	for _, check := range report.Checks {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", check.Status, check.Severity, check.Category, check.ID, check.Message)
	}
	fmt.Fprintf(tw, "\npassed=%d warnings=%d failed=%d skipped=%d\n", report.Summary.Passed, report.Summary.Warnings, report.Summary.Failed, report.Summary.Skipped)
	return tw.Flush()
}

func RenderMarkdown(w io.Writer, report Report) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# IronRoot Security Check Report\n\n")
	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "- Passed: %d\n- Warnings: %d\n- Failed: %d\n- Skipped: %d\n\n", report.Summary.Passed, report.Summary.Warnings, report.Summary.Failed, report.Summary.Skipped)
	fmt.Fprintf(&b, "## Checks\n\n")
	fmt.Fprintf(&b, "| Status | Severity | Category | ID | Message | Remediation |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- | --- | --- |\n")
	for _, check := range report.Checks {
		fmt.Fprintf(&b, "| %s | %s | %s | `%s` | %s | %s |\n",
			escapeMD(string(check.Status)),
			escapeMD(string(check.Severity)),
			escapeMD(check.Category),
			escapeMD(check.ID),
			escapeMD(check.Message),
			escapeMD(check.Remediation),
		)
	}
	_, err := w.Write(b.Bytes())
	return err
}

func escapeMD(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
