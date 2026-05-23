package securitycheck

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type BootstrapOptions struct {
	NonInteractive  bool
	AcknowledgeRisk bool
	OutputChecklist string
	In              io.Reader
	Out             io.Writer
}

type ChecklistItem struct {
	Section string
	Text    string
}

var BootstrapChecklist = []ChecklistItem{
	{"Offline Root CA checklist", "Root CA private key is offline or air-gapped."},
	{"Offline Root CA checklist", "Root CA private key is encrypted at rest."},
	{"Offline Root CA checklist", "Root CA private key has at least two secure offline backups."},
	{"Offline Root CA checklist", "Root CA is used only to sign Intermediate CA certificates."},
	{"Offline Root CA checklist", "Normal server certificates are never signed directly by the Root CA."},
	{"Intermediate CA checklist", "Intermediate CA is signed by the offline Root CA."},
	{"Intermediate CA checklist", "Intermediate CA private key is encrypted at rest."},
	{"Intermediate CA checklist", "Intermediate CA private key is stored only on the IronRoot server."},
	{"Intermediate CA checklist", "Intermediate CA lifetime is shorter than Root CA lifetime, recommended 5 years."},
	{"Server hardening checklist", "IronRoot server filesystem permissions are restricted."},
	{"Server hardening checklist", "IronRoot runs as a non-root user."},
	{"Server hardening checklist", "Admin access is protected."},
	{"Backup checklist", "Backup plan exists for SQLite data or future PostgreSQL data."},
	{"Backup checklist", "Backup plan exists for Intermediate CA material."},
	{"API TLS checklist", "TLS is enabled for the IronRoot API."},
	{"API TLS checklist", "API bind address is intentional and protected."},
	{"Observability checklist", "Audit logging is enabled."},
	{"Observability checklist", "OpenTelemetry is configured or intentionally disabled."},
	{"Migration readiness checklist", "CA generation metadata is tracked for future root migration."},
	{"Recovery checklist", "Recovery steps are documented and tested without exposing Root CA private keys."},
}

func RunBootstrapGuide(opts BootstrapOptions) error {
	in := opts.In
	if in == nil {
		in = os.Stdin
	}
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	if opts.NonInteractive && !opts.AcknowledgeRisk {
		return fmt.Errorf("--non-interactive requires --acknowledge-risk so automation records deliberate acceptance of bootstrap assumptions")
	}
	fmt.Fprintln(out, "IronRoot first-run security bootstrap")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Offline Root CA: create it on an offline or air-gapped machine. The Root CA private key must never be copied to the online IronRoot server. Encrypt it at rest, back it up to at least two offline locations, and use it only to sign Intermediate CA certificates. Recommended Root CA lifetime: 20 years.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Intermediate CA: this is the online issuing CA. Its private key may live on the IronRoot server, but it must be encrypted at rest and protected with strict filesystem permissions. Recommended Intermediate CA lifetime: 5 years. Normal server certificates should be signed by the Intermediate CA and default to 90 days.")
	fmt.Fprintln(out)
	if !opts.NonInteractive {
		reader := bufio.NewReader(in)
		for _, item := range BootstrapChecklist {
			fmt.Fprintf(out, "%s\n%s\nConfirm [y/N]: ", item.Section, item.Text)
			answer, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				if !opts.AcknowledgeRisk {
					return fmt.Errorf("bootstrap stopped: %q was not confirmed; pass --acknowledge-risk only when intentionally accepting this risk", item.Text)
				}
				fmt.Fprintln(out, "Risk acknowledged; continuing.")
			}
			fmt.Fprintln(out)
		}
	} else {
		fmt.Fprintln(out, "Non-interactive mode enabled; risk acknowledgement recorded by flag.")
	}
	if opts.OutputChecklist != "" {
		if err := os.WriteFile(opts.OutputChecklist, []byte(SecurityChecklistMarkdown()), 0o600); err != nil {
			return err
		}
		fmt.Fprintln(out, "Wrote security checklist:", opts.OutputChecklist)
	}
	fmt.Fprintln(out, "Bootstrap guide complete. Run `ironroot-admin security-check` before exposing the API.")
	return nil
}

func SecurityChecklistMarkdown() string {
	var b strings.Builder
	b.WriteString("# IronRoot Security Bootstrap Checklist\n\n")
	current := ""
	for _, item := range BootstrapChecklist {
		if item.Section != current {
			current = item.Section
			b.WriteString("## " + current + "\n\n")
		}
		b.WriteString("- [ ] " + item.Text + "\n")
	}
	return b.String()
}
