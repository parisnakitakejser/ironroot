package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"

	"github.com/ironroot/ironroot/internal/ca"
	"github.com/ironroot/ironroot/internal/config"
	ironcrypto "github.com/ironroot/ironroot/internal/crypto"
	"github.com/ironroot/ironroot/internal/db"
	"github.com/ironroot/ironroot/internal/securitycheck"
	"github.com/ironroot/ironroot/internal/telemetry"
	apiclient "github.com/ironroot/ironroot/pkg/client"
)

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e ExitError) Unwrap() error { return e.Err }
func (e ExitError) ExitCode() int { return e.Code }

func New() *cobra.Command {
	var server, configPath string
	cmd := &cobra.Command{
		Use:          "ironroot-admin",
		Short:        "Admin CLI for IronRoot PKI",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			slog.SetDefault(telemetry.NewLogger(os.Stderr, "info"))
			_, _ = telemetry.Configure(cmd.Context(), config.Default().Telemetry, "ironroot-admin")
		},
	}
	cmd.PersistentFlags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "IronRoot config file")
	cmd.AddCommand(caCommands(), initServer(&configPath), importIntermediate(&configPath), createToken(&configPath), bootstrap(), securityCheck(), apiCommand("list-tokens", &server, func(ctx context.Context, c *apiclient.Client) error {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "list-tokens is local-store only in this MVP; use server DB tooling"})
	}), apiCommand("list-certs", &server, func(ctx context.Context, c *apiclient.Client) error {
		entries, err := c.Audit(ctx)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(entries)
	}), revokeCert(&server), migrationStatus(&server))
	cmd.AddCommand(&cobra.Command{Use: "revoke-token", Short: "Revoke a bootstrap token by id", RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("revoke-token requires direct store access and is reserved for the server-side admin plane")
	}})
	cmd.AddCommand(&cobra.Command{Use: "rotate-intermediate", Short: "Import a new active intermediate generation", RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("rotate-intermediate design is documented; API mutation is intentionally deferred")
	}})
	return cmd
}

func caCommands() *cobra.Command {
	cmd := &cobra.Command{Use: "ca", Short: "Create, inspect, and verify local IronRoot CA material"}
	cmd.AddCommand(createRootCA(), createIntermediateCA(), inspectCA(), verifyChainCA())
	return cmd
}

func createRootCA() *cobra.Command {
	var name, out, password, passwordFile, lifetime string
	cmd := &cobra.Command{Use: "create-root", Short: "Generate an encrypted offline Root CA key and certificate", RunE: func(cmd *cobra.Command, args []string) error {
		pass, passPath, err := resolvePassword(password, passwordFile, filepath.Join(out, "root-ca.password"))
		if err != nil {
			return err
		}
		dur, err := parseDaysDuration(lifetime)
		if err != nil {
			return err
		}
		res, err := ca.CreateRoot(ca.CreateRootOptions{Name: name, OutDir: out, Password: pass, Lifetime: dur})
		if err != nil {
			return err
		}
		res.PasswordFile = passPath
		fmt.Fprintf(cmd.OutOrStdout(), "Root CA created successfully.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  certificate: %s\n  private key: %s\n  fingerprint: %s\n  expires: %s\n", res.CertPath, res.KeyPath, res.Fingerprint, res.NotAfter.Format(time.RFC3339))
		if passPath != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  password file: %s\n", passPath)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nOffline-root guidance: keep %s off the IronRoot server, out of containers, and out of Kubernetes.\n", res.KeyPath)
		passwordHint := "--root-password <root-password>"
		if passPath != "" {
			passwordHint = "--root-password-file " + passPath
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Next step:\n  ironroot-admin ca create-intermediate --root-cert %s --root-key %s %s --out ./pki/intermediate\n", res.CertPath, res.KeyPath, passwordHint)
		return nil
	}}
	cmd.Flags().StringVar(&name, "name", "IronRoot Local Root CA", "Root CA common name")
	cmd.Flags().StringVar(&out, "out", "./pki/root", "output directory")
	cmd.Flags().StringVar(&password, "password", "", "Root CA key password; omit to generate a local password file")
	cmd.Flags().StringVar(&passwordFile, "password-file", "", "file containing Root CA key password")
	cmd.Flags().StringVar(&lifetime, "lifetime", "20y", "Root CA lifetime, for example 20y or 175200h")
	return cmd
}

func createIntermediateCA() *cobra.Command {
	var name, out, rootCert, rootKey, rootPassword, rootPasswordFile, password, passwordFile, lifetime string
	cmd := &cobra.Command{Use: "create-intermediate", Short: "Generate an encrypted Intermediate CA and sign it with the Root CA", RunE: func(cmd *cobra.Command, args []string) error {
		rootPass, _, err := resolveExistingPassword(rootPassword, rootPasswordFile, filepath.Join(filepath.Dir(rootKey), "root-ca.password"))
		if err != nil {
			return err
		}
		pass, passPath, err := resolvePassword(password, passwordFile, filepath.Join(out, "intermediate-ca.password"))
		if err != nil {
			return err
		}
		dur, err := parseDaysDuration(lifetime)
		if err != nil {
			return err
		}
		res, err := ca.CreateIntermediate(ca.CreateIntermediateOptions{
			Name: name, OutDir: out, RootCertPath: rootCert, RootKeyPath: rootKey, RootPassword: rootPass, Password: pass, Lifetime: dur,
		})
		if err != nil {
			return err
		}
		res.PasswordFile = passPath
		fmt.Fprintf(cmd.OutOrStdout(), "Intermediate CA created successfully.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  certificate: %s\n  private key: %s\n  csr: %s\n  chain: %s\n  fingerprint: %s\n  expires: %s\n", res.CertPath, res.KeyPath, res.CSRPath, res.ChainPath, res.Fingerprint, res.NotAfter.Format(time.RFC3339))
		if passPath != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  password file: %s\n", passPath)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nCopy the Intermediate CA certificate, encrypted key, chain, and password into the IronRoot server PKI path. Do not copy the Root CA private key.\n")
		return nil
	}}
	cmd.Flags().StringVar(&name, "name", "IronRoot Local Intermediate CA", "Intermediate CA common name")
	cmd.Flags().StringVar(&out, "out", "./pki/intermediate", "output directory")
	cmd.Flags().StringVar(&rootCert, "root-cert", "./pki/root/root-ca.crt", "Root CA certificate")
	cmd.Flags().StringVar(&rootKey, "root-key", "./pki/root/root-ca.key", "encrypted Root CA private key")
	cmd.Flags().StringVar(&rootPassword, "root-password", "", "Root CA key password")
	cmd.Flags().StringVar(&rootPasswordFile, "root-password-file", "", "file containing Root CA key password")
	cmd.Flags().StringVar(&password, "password", "", "Intermediate CA key password; omit to generate a local password file")
	cmd.Flags().StringVar(&passwordFile, "password-file", "", "file containing Intermediate CA key password")
	cmd.Flags().StringVar(&lifetime, "lifetime", "5y", "Intermediate CA lifetime, for example 5y or 43800h")
	return cmd
}

func inspectCA() *cobra.Command {
	var format string
	cmd := &cobra.Command{Use: "inspect <cert> [cert...]", Short: "Inspect CA and certificate PEM files", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		info, err := ca.InspectCertificates(args)
		if err != nil {
			return err
		}
		if format == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
		}
		for _, cert := range info {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n  subject: %s\n  issuer: %s\n  ca: %t\n  fingerprint: %s\n  expires: %s\n", cert.Path, cert.Subject, cert.Issuer, cert.IsCA, cert.Fingerprint, cert.NotAfter.Format(time.RFC3339))
		}
		return nil
	}}
	cmd.Flags().StringVar(&format, "output", "table", "output format: table or json")
	return cmd
}

func verifyChainCA() *cobra.Command {
	var root, intermediate, leaf string
	cmd := &cobra.Command{Use: "verify-chain", Short: "Verify Root, Intermediate, and optional leaf certificate chain", RunE: func(cmd *cobra.Command, args []string) error {
		if err := ca.VerifyChain(root, intermediate, leaf); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Certificate chain verified successfully.")
		return nil
	}}
	cmd.Flags().StringVar(&root, "root-cert", "./pki/root/root-ca.crt", "Root CA certificate")
	cmd.Flags().StringVar(&intermediate, "intermediate-cert", "./pki/intermediate/intermediate-ca.crt", "Intermediate CA certificate")
	cmd.Flags().StringVar(&leaf, "cert", "", "optional leaf certificate to verify")
	return cmd
}

func bootstrap() *cobra.Command {
	var configPath, outputChecklist string
	var nonInteractive, acknowledgeRisk bool
	cmd := &cobra.Command{Use: "bootstrap", Short: "Guide first-time operators through secure IronRoot setup", RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		ctx, span := otel.Tracer("ironroot-admin").Start(cmd.Context(), "ironroot-admin bootstrap")
		defer span.End()
		if _, err := config.Load(configPath); err != nil {
			telemetry.RecordCommand(ctx, "bootstrap", start, err)
			return ExitError{Code: 2, Err: err}
		}
		err := securitycheck.RunBootstrapGuide(securitycheck.BootstrapOptions{
			NonInteractive:  nonInteractive,
			AcknowledgeRisk: acknowledgeRisk,
			OutputChecklist: outputChecklist,
			In:              cmd.InOrStdin(),
			Out:             cmd.OutOrStdout(),
		})
		telemetry.RecordCommand(ctx, "bootstrap", start, err)
		if err != nil {
			return ExitError{Code: 2, Err: err}
		}
		return nil
	}}
	cmd.Flags().StringVar(&configPath, "config", "", "IronRoot config file")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "run without prompts for automation")
	cmd.Flags().BoolVar(&acknowledgeRisk, "acknowledge-risk", false, "deliberately acknowledge bootstrap assumptions or unconfirmed prompts")
	cmd.Flags().StringVar(&outputChecklist, "output-checklist", "", "write Markdown bootstrap checklist")
	return cmd
}

func securityCheck() *cobra.Command {
	var configPath, outputValue, failOnValue, writeReport string
	cmd := &cobra.Command{Use: "security-check", Short: "Run IronRoot host, PKI, config, runtime, and observability checks", RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		ctx, span := otel.Tracer("ironroot-admin").Start(cmd.Context(), "ironroot-admin security-check")
		defer span.End()
		cfg, err := config.Load(configPath)
		if err != nil {
			telemetry.RecordCommand(ctx, "security-check", start, err)
			return ExitError{Code: 2, Err: err}
		}
		format, err := securitycheck.ParseOutputFormat(outputValue)
		if err != nil {
			telemetry.RecordCommand(ctx, "security-check", start, err)
			return ExitError{Code: 2, Err: err}
		}
		failOn, err := securitycheck.ParseSeverity(failOnValue)
		if err != nil {
			telemetry.RecordCommand(ctx, "security-check", start, err)
			return ExitError{Code: 2, Err: err}
		}
		report := securitycheck.DefaultRunner().Run(ctx, securitycheck.Target{Config: cfg, ConfigPath: configPath, Now: time.Now().UTC()})
		if err := securitycheck.Render(cmd.OutOrStdout(), report, format); err != nil {
			telemetry.RecordCommand(ctx, "security-check", start, err)
			return ExitError{Code: 2, Err: err}
		}
		if writeReport != "" {
			f, err := os.Create(writeReport)
			if err != nil {
				telemetry.RecordCommand(ctx, "security-check", start, err)
				return ExitError{Code: 2, Err: err}
			}
			defer f.Close()
			reportFormat := securitycheck.OutputMarkdown
			if format == securitycheck.OutputJSON {
				reportFormat = securitycheck.OutputJSON
			}
			if err := securitycheck.Render(f, report, reportFormat); err != nil {
				telemetry.RecordCommand(ctx, "security-check", start, err)
				return ExitError{Code: 2, Err: err}
			}
		}
		slog.InfoContext(ctx, "security-check summary", "passed", report.Summary.Passed, "warnings", report.Summary.Warnings, "failed", report.Summary.Failed, "skipped", report.Summary.Skipped)
		var exitErr error
		if securitycheck.FailsThreshold(report.Checks, failOn) {
			exitErr = ExitError{Code: 1, Err: fmt.Errorf("security-check found failures at or above %s", failOn)}
		}
		telemetry.RecordCommand(ctx, "security-check", start, exitErr)
		return exitErr
	}}
	cmd.Flags().StringVar(&configPath, "config", "", "IronRoot config file")
	cmd.Flags().StringVar(&outputValue, "output", "table", "output format: table, json, markdown")
	cmd.Flags().StringVar(&failOnValue, "fail-on", "critical", "fail when failed checks are at or above severity: info, low, medium, high, critical")
	cmd.Flags().StringVar(&writeReport, "write-report", "", "write a report file; markdown by default, json when --output json")
	return cmd
}

func initServer(configPath *string) *cobra.Command {
	return &cobra.Command{Use: "init-server", Short: "Initialize server configuration and migrations", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		store, err := db.Open(cmd.Context(), cfg.Database)
		if err != nil {
			return err
		}
		defer store.Close()
		return store.Migrate(cmd.Context())
	}}
}

func importIntermediate(configPath *string) *cobra.Command {
	return &cobra.Command{Use: "import-intermediate", Short: "Validate mounted intermediate CA material", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		_, err = os.Stat(cfg.PKI.IntermediateCertFile)
		return err
	}}
}

func createToken(configPath *string) *cobra.Command {
	var hostname, ttl string
	cmd := &cobra.Command{Use: "create-token", Short: "Create a hashed bootstrap token in the configured SQLite database", RunE: func(cmd *cobra.Command, args []string) error {
		if hostname == "" {
			return fmt.Errorf("--host is required")
		}
		dur, err := time.ParseDuration(ttl)
		if err != nil {
			return err
		}
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		store, err := db.Open(cmd.Context(), cfg.Database)
		if err != nil {
			return err
		}
		defer store.Close()
		if err := store.Migrate(cmd.Context()); err != nil {
			return err
		}
		token, err := ironcrypto.NewBootstrapToken()
		if err != nil {
			return err
		}
		err = store.CreateBootstrapToken(cmd.Context(), db.BootstrapToken{ID: uuid.NewString(), TokenHash: ironcrypto.HashToken(token), Hostname: hostname, ExpiresAt: time.Now().UTC().Add(dur), CreatedAt: time.Now().UTC()})
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"token": token, "hostname": hostname, "ttl": ttl})
	}}
	cmd.Flags().StringVar(&hostname, "host", "", "hostname allowed to enroll")
	cmd.Flags().StringVar(&ttl, "ttl", "1h", "token lifetime")
	return cmd
}

func revokeCert(server *string) *cobra.Command {
	var serial, reason string
	cmd := apiCommand("revoke-cert", server, func(ctx context.Context, c *apiclient.Client) error {
		return c.RevokeCertificate(ctx, serial, reason)
	})
	cmd.Flags().StringVar(&serial, "serial", "", "certificate serial")
	cmd.Flags().StringVar(&reason, "reason", "admin-request", "revocation reason")
	return cmd
}

func migrationStatus(server *string) *cobra.Command {
	return apiCommand("migration-status", server, func(ctx context.Context, c *apiclient.Client) error {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "multiple CA generations supported in schema; promotion API deferred"})
	})
}

func apiCommand(use string, server *string, run func(context.Context, *apiclient.Client) error) *cobra.Command {
	return &cobra.Command{Use: use, RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		err := run(cmd.Context(), apiclient.New(*server))
		telemetry.RecordCommand(cmd.Context(), use, start, err)
		return err
	}}
}

func resolvePassword(value, file, defaultFile string) (string, string, error) {
	if value != "" {
		return value, "", nil
	}
	if file != "" {
		pass, err := readPasswordFile(file)
		return pass, file, err
	}
	pass, err := randomPassword()
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(defaultFile), 0o700); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(defaultFile, []byte(pass+"\n"), 0o600); err != nil {
		return "", "", err
	}
	return pass, defaultFile, nil
}

func resolveExistingPassword(value, file, defaultFile string) (string, string, error) {
	if value != "" {
		return value, "", nil
	}
	if file == "" {
		file = defaultFile
	}
	pass, err := readPasswordFile(file)
	return pass, file, err
}

func readPasswordFile(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	pass := strings.TrimSpace(string(body))
	if pass == "" {
		return "", fmt.Errorf("password file %s is empty", path)
	}
	return pass, nil
}

func randomPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func parseDaysDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "y") {
		years, err := time.ParseDuration(strings.TrimSuffix(value, "y") + "h")
		if err != nil {
			return 0, err
		}
		return years * 365 * 24, nil
	}
	return time.ParseDuration(value)
}
