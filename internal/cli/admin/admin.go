package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
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
	cmd.AddCommand(caCommands(), initServer(&configPath), importIntermediate(&configPath), createToken(&configPath), listTokens(&configPath), revokeToken(&configPath), inspectToken(&configPath), bootstrap(), securityCheck(), apiCommand("list-certs", &server, func(ctx context.Context, c *apiclient.Client) error {
		entries, err := c.Audit(ctx)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(entries)
	}), revokeCert(&server), migrationStatus(&server))
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
	var (
		name, commonName, organization, organizationalUnit, country, province, locality string
		algorithm, curve, keyFormat, out, password, passwordEnv, passwordFile           string
		validity, notBeforeValue                                                        string
		rsaBits, maxPathLength                                                          int
		encryptKey, isCA, allowCertSigning, allowCRLSigning                             bool
		allowServerAuth, allowClientAuth, allowCodeSigning, allowEmailProtection        bool
		generateTrustBundle, writePEM, writeDER, offline, backupReminder                bool
		crlEnabled, ocspEnabled                                                         bool
		permitDNS, permitIP, permitEmail                                                []string
	)
	cmd := &cobra.Command{
		Use:   "create-root",
		Short: "Generate an offline Root CA key and certificate",
		Long: "Generate Root CA material for IronRoot. Beginner mode uses secure defaults: ECDSA P-384, " +
			"20 year validity, encrypted private key, CA signing only, and max path length 1.",
		Example: `  ironroot-admin ca create-root \
    --name "IronRoot Local Root CA" \
    --out .localdev/pki/root

  ROOT_CA_PASSWORD="$(openssl rand -base64 32)" ironroot-admin ca create-root \
    --name "IronRoot Production Root CA" \
    --common-name "IronRoot Root CA" \
    --organization "IronRoot" \
    --organizational-unit "Security" \
    --country "DK" \
    --province "Hovedstaden" \
    --locality "Copenhagen" \
    --algorithm ecdsa \
    --curve p384 \
    --validity 20y \
    --max-path-length 1 \
    --key-password-env ROOT_CA_PASSWORD \
    --encrypt-key \
    --offline \
		--out ./pki/root`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pass, passPath, err := resolveRootPassword(password, passwordEnv, passwordFile, filepath.Join(out, "root-ca.password"), encryptKey)
			if err != nil {
				return err
			}
			dur, err := parseDaysDuration(validity)
			if err != nil {
				return err
			}
			var notBefore time.Time
			if notBeforeValue != "" {
				notBefore, err = time.Parse(time.RFC3339, notBeforeValue)
				if err != nil {
					return fmt.Errorf("--not-before must use RFC3339 format, for example 2026-01-02T15:04:05Z: %w", err)
				}
			}
			if !encryptKey {
				fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: --encrypt-key=false writes the Root CA private key unencrypted. Use only for throwaway local testing.")
			}
			if allowServerAuth || allowClientAuth || allowCodeSigning || allowEmailProtection {
				fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: Root CAs should normally not include leaf certificate usages such as serverAuth, clientAuth, codeSigning, or emailProtection.")
			}
			if !offline {
				fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: --offline=false documents an online Root workflow. Production Root CA private keys should remain offline.")
			}
			res, err := ca.CreateRoot(ca.CreateRootOptions{
				Name: name, CommonName: commonName, Organization: organization, OrganizationalUnit: organizationalUnit,
				Country: country, Province: province, Locality: locality, Algorithm: algorithm, RSABits: rsaBits,
				Curve: curve, KeyFormat: keyFormat, EncryptKey: encryptKey, Password: pass, Lifetime: dur,
				NotBefore: notBefore, MaxPathLength: maxPathLength, IsCA: isCA, AllowCertSigning: allowCertSigning,
				AllowCRLSigning: allowCRLSigning, AllowServerAuth: allowServerAuth, AllowClientAuth: allowClientAuth,
				AllowCodeSigning: allowCodeSigning, AllowEmailProtection: allowEmailProtection,
				GenerateTrustBundle: generateTrustBundle, WritePEM: writePEM, WriteDER: writeDER, Offline: offline,
				BackupReminder: backupReminder, CRLEnabled: crlEnabled, OCSPEnabled: ocspEnabled,
				PermittedDNSDomains: permitDNS, PermittedIPRanges: permitIP, PermittedEmailDomains: permitEmail,
				OutDir: out,
			})
			if err != nil {
				return err
			}
			res.PasswordFile = passPath
			fmt.Fprintf(cmd.OutOrStdout(), "Root CA created successfully.\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  certificate: %s\n  private key: %s\n  public key: %s\n  metadata: %s\n  fingerprints: %s\n  recovery notes: %s\n", res.CertPath, res.KeyPath, res.PublicKeyPath, res.MetadataPath, res.FingerprintsPath, res.RecoveryPath)
			if res.PEMPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  pem export: %s\n", res.PEMPath)
			}
			if res.DERPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  der export: %s\n", res.DERPath)
			}
			if res.TrustBundleDir != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  trust bundle: %s\n", res.TrustBundleDir)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  fingerprint: %s\n  expires: %s\n", res.Fingerprint, res.NotAfter.Format(time.RFC3339))
			if passPath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  password file: %s\n", passPath)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\nIMPORTANT: keep %s offline, encrypted, out of containers, out of Kubernetes, and out of CI.\n", res.KeyPath)
			passwordHint := "--root-password <root-password>"
			if passPath != "" {
				passwordHint = "--root-password-file " + passPath
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Next step:\n  ironroot-admin ca create-intermediate --root-cert %s --root-key %s %s --out ./pki/intermediate\n", res.CertPath, res.KeyPath, passwordHint)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "IronRoot Local Root CA", "friendly Root CA name; default is recommended for local quick starts")
	cmd.Flags().StringVar(&commonName, "common-name", "", "certificate subject common name; defaults to --name")
	cmd.Flags().StringVar(&organization, "organization", "IronRoot", "certificate subject organization")
	cmd.Flags().StringVar(&organizationalUnit, "organizational-unit", "", "certificate subject organizational unit")
	cmd.Flags().StringVar(&country, "country", "", "certificate subject country code, for example DK")
	cmd.Flags().StringVar(&province, "province", "", "certificate subject state or province")
	cmd.Flags().StringVar(&locality, "locality", "", "certificate subject locality or city")
	cmd.Flags().StringVar(&algorithm, "algorithm", "ecdsa", "Root key algorithm: ecdsa or rsa; ecdsa is recommended")
	cmd.Flags().IntVar(&rsaBits, "rsa-bits", 4096, "RSA key size when --algorithm rsa; 4096 is recommended")
	cmd.Flags().StringVar(&curve, "curve", "p384", "ECDSA curve when --algorithm ecdsa: p256, p384, or p521; p384 is recommended for Root CAs")
	cmd.Flags().StringVar(&keyFormat, "key-format", "pkcs8", "private key format; pkcs8 is recommended")
	cmd.Flags().BoolVar(&encryptKey, "encrypt-key", true, "encrypt Root CA private key at rest; production should keep this true")
	cmd.Flags().StringVar(&password, "key-password", "", "Root CA key password; avoid shell history in production")
	cmd.Flags().StringVar(&passwordEnv, "key-password-env", "", "environment variable containing Root CA key password")
	cmd.Flags().StringVar(&passwordFile, "key-password-file", "", "file containing Root CA key password")
	cmd.Flags().StringVar(&validity, "validity", "20y", "Root CA validity, for example 20y or 175200h")
	cmd.Flags().StringVar(&notBeforeValue, "not-before", "", "certificate notBefore timestamp in RFC3339; defaults to current time minus one minute")
	cmd.Flags().IntVar(&maxPathLength, "max-path-length", 1, "maximum CA path length below Root; 1 allows Intermediate CAs but not deeper hierarchies")
	cmd.Flags().BoolVar(&isCA, "is-ca", true, "generate a CA certificate; Root CA creation requires true")
	cmd.Flags().BoolVar(&allowCertSigning, "allow-cert-signing", true, "allow Root CA to sign Intermediate CA certificates")
	cmd.Flags().BoolVar(&allowCRLSigning, "allow-crl-signing", true, "allow Root CA to sign certificate revocation lists")
	cmd.Flags().BoolVar(&allowServerAuth, "allow-server-auth", false, "include serverAuth extended usage; not recommended for Root CAs")
	cmd.Flags().BoolVar(&allowClientAuth, "allow-client-auth", false, "include clientAuth extended usage; not recommended for Root CAs")
	cmd.Flags().BoolVar(&allowCodeSigning, "allow-code-signing", false, "include codeSigning extended usage; not recommended for Root CAs")
	cmd.Flags().BoolVar(&allowEmailProtection, "allow-email-protection", false, "include emailProtection extended usage; not recommended for Root CAs")
	cmd.Flags().BoolVar(&generateTrustBundle, "generate-trust-bundle", true, "write trust-bundle/root-ca.crt for distribution")
	cmd.Flags().BoolVar(&writePEM, "pem", true, "write root-ca.pem export")
	cmd.Flags().BoolVar(&writeDER, "der", true, "write root-ca.der export")
	cmd.Flags().BoolVar(&offline, "offline", true, "record that this Root CA is intended for offline or air-gapped custody")
	cmd.Flags().BoolVar(&backupReminder, "backup-reminder", true, "include backup and recovery reminders in recovery.txt")
	cmd.Flags().BoolVar(&crlEnabled, "crl-enabled", false, "record future CRL support intent in metadata")
	cmd.Flags().BoolVar(&ocspEnabled, "ocsp-enabled", false, "record future OCSP support intent in metadata")
	cmd.Flags().StringSliceVar(&permitDNS, "permit-dns", nil, "permitted DNS domain constraint; advanced Root CA name constraint")
	cmd.Flags().StringSliceVar(&permitIP, "permit-ip", nil, "permitted IP CIDR constraint; advanced Root CA name constraint")
	cmd.Flags().StringSliceVar(&permitEmail, "permit-email", nil, "permitted email domain constraint; advanced Root CA name constraint")
	cmd.Flags().StringVar(&out, "out", "./pki/root", "output directory")
	cmd.Flags().StringVar(&password, "password", "", "deprecated alias for --key-password")
	cmd.Flags().StringVar(&passwordFile, "password-file", "", "deprecated alias for --key-password-file")
	cmd.Flags().StringVar(&validity, "lifetime", "20y", "deprecated alias for --validity")
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
		switch format {
		case "json":
			return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
		case "markdown":
			for _, cert := range info {
				fmt.Fprintf(cmd.OutOrStdout(), "## %s\n\n", cert.Path)
				fmt.Fprintf(cmd.OutOrStdout(), "| Field | Value |\n|---|---|\n")
				fmt.Fprintf(cmd.OutOrStdout(), "| Subject | `%s` |\n| Issuer | `%s` |\n| Algorithm | `%s` |\n| Curve | `%s` |\n| Key size | `%d` |\n| CA | `%t` |\n| Max path length | `%d` |\n| Key usage | `%s` |\n| Extended key usage | `%s` |\n| Not before | `%s` |\n| Not after | `%s` |\n| SHA-256 fingerprint | `%s` |\n\n", cert.Subject, cert.Issuer, cert.Algorithm, cert.Curve, cert.KeySize, cert.IsCA, cert.MaxPathLength, strings.Join(cert.KeyUsage, ", "), strings.Join(cert.ExtKeyUsage, ", "), cert.NotBefore.Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339), cert.Fingerprint)
			}
			return nil
		case "table":
			for _, cert := range info {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n  subject: %s\n  issuer: %s\n  algorithm: %s\n  curve: %s\n  key size: %d\n  ca: %t\n  max path length: %d\n  key usage: %s\n  extended key usage: %s\n  fingerprint: %s\n  valid: %s to %s\n", cert.Path, cert.Subject, cert.Issuer, cert.Algorithm, cert.Curve, cert.KeySize, cert.IsCA, cert.MaxPathLength, strings.Join(cert.KeyUsage, ", "), strings.Join(cert.ExtKeyUsage, ", "), cert.Fingerprint, cert.NotBefore.Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339))
			}
			return nil
		default:
			return fmt.Errorf("unsupported output format %q; use table, json, or markdown", format)
		}
	}}
	cmd.Flags().StringVar(&format, "output", "table", "output format: table, json, or markdown")
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
		now := time.Now().UTC()
		id := uuid.NewString()
		expires := now.Add(dur)
		err = store.CreateBootstrapToken(cmd.Context(), db.BootstrapToken{ID: id, TokenHash: ironcrypto.HashToken(token), Hostname: hostname, ExpiresAt: expires, CreatedAt: now})
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Bootstrap token created successfully.")
		fmt.Fprintf(cmd.OutOrStdout(), "\nID:\n%s\n\nHost:\n%s\n\nExpires:\n%s\n\n", id, hostname, expires.Format(time.RFC3339))
		if dur > 24*time.Hour {
			fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: bootstrap token TTL is longer than 24h. Short-lived tokens are safer.")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Token:\n%s\n\n", token)
		fmt.Fprintln(cmd.OutOrStdout(), "Next step:")
		fmt.Fprintf(cmd.OutOrStdout(), "ironroot-client enroll \\\n  --server http://localhost:8443 \\\n  --hostname %s \\\n  --token %s\n", hostname, token)
		return nil
	}}
	cmd.Flags().StringVar(&hostname, "host", "", "hostname allowed to enroll")
	cmd.Flags().StringVar(&ttl, "ttl", "1h", "token lifetime")
	return cmd
}

type tokenRow struct {
	ID             string     `json:"id"`
	Hostname       string     `json:"hostname"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	Used           bool       `json:"used"`
	UsageCount     int        `json:"usage_count"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	EnrollmentID   string     `json:"enrollment_id,omitempty"`
	EnrollmentHost string     `json:"enrollment_host,omitempty"`
}

func listTokens(configPath *string) *cobra.Command {
	var active, expired, revoked, used, unused, wide bool
	var host string
	var jsonOut, markdown bool
	cmd := &cobra.Command{Use: "list-tokens", Short: "List bootstrap tokens from the configured store", RunE: func(cmd *cobra.Command, args []string) error {
		store, closeFn, err := openAdminStore(cmd.Context(), *configPath)
		if err != nil {
			return err
		}
		defer closeFn()
		rows, err := loadTokenRows(cmd.Context(), store)
		if err != nil {
			return err
		}
		rows = filterTokenRows(rows, tokenFilters{active: active, expired: expired, revoked: revoked, used: used, unused: unused, host: host})
		switch {
		case jsonOut:
			return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
		case markdown:
			renderTokenMarkdown(cmd.OutOrStdout(), rows, wide)
		default:
			renderTokenTable(cmd.OutOrStdout(), rows, wide)
		}
		return nil
	}}
	cmd.Flags().BoolVar(&active, "active", false, "show active tokens")
	cmd.Flags().BoolVar(&expired, "expired", false, "show expired tokens")
	cmd.Flags().BoolVar(&revoked, "revoked", false, "show revoked tokens")
	cmd.Flags().BoolVar(&used, "used", false, "show used tokens")
	cmd.Flags().BoolVar(&unused, "unused", false, "show unused tokens")
	cmd.Flags().StringVar(&host, "host", "", "filter by hostname")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render JSON output")
	cmd.Flags().BoolVar(&markdown, "markdown", false, "render Markdown output")
	cmd.Flags().BoolVar(&wide, "wide", false, "include enrollment relationship columns")
	return cmd
}

func inspectToken(configPath *string) *cobra.Command {
	var jsonOut, markdown bool
	cmd := &cobra.Command{Use: "inspect-token <token-id>", Short: "Inspect one bootstrap token without revealing the token secret", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		store, closeFn, err := openAdminStore(cmd.Context(), *configPath)
		if err != nil {
			return err
		}
		defer closeFn()
		row, err := loadTokenRow(cmd.Context(), store, args[0])
		if err != nil {
			return err
		}
		switch {
		case jsonOut:
			return json.NewEncoder(cmd.OutOrStdout()).Encode(row)
		case markdown:
			renderTokenMarkdown(cmd.OutOrStdout(), []tokenRow{row}, true)
		default:
			renderTokenTable(cmd.OutOrStdout(), []tokenRow{row}, true)
		}
		if row.Status == "expired" && !row.Used {
			fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: token expired unused. Consider revoking or cleaning it up.")
		}
		return nil
	}}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "render JSON output")
	cmd.Flags().BoolVar(&markdown, "markdown", false, "render Markdown output")
	return cmd
}

func revokeToken(configPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "revoke-token <token-id>", Short: "Revoke a bootstrap token by id", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		store, closeFn, err := openAdminStore(cmd.Context(), *configPath)
		if err != nil {
			return err
		}
		defer closeFn()
		if err := store.RevokeBootstrapToken(cmd.Context(), args[0]); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Bootstrap token revoked: %s\n", args[0])
		return nil
	}}
	return cmd
}

func openAdminStore(ctx context.Context, configPath string) (*db.SQLStore, func(), error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, err
	}
	store, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return nil, nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	return store, func() { _ = store.Close() }, nil
}

func loadTokenRows(ctx context.Context, store *db.SQLStore) ([]tokenRow, error) {
	tokens, err := store.ListBootstrapTokens(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]tokenRow, 0, len(tokens))
	for _, token := range tokens {
		row, err := tokenRowFrom(ctx, store, token)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func loadTokenRow(ctx context.Context, store *db.SQLStore, id string) (tokenRow, error) {
	token, err := store.GetBootstrapToken(ctx, id)
	if err != nil {
		return tokenRow{}, err
	}
	return tokenRowFrom(ctx, store, token)
}

func tokenRowFrom(ctx context.Context, store *db.SQLStore, token db.BootstrapToken) (tokenRow, error) {
	usage, err := store.BootstrapTokenUsage(ctx, token.ID)
	if err != nil {
		return tokenRow{}, err
	}
	row := tokenRow{
		ID: token.ID, Hostname: token.Hostname, CreatedAt: token.CreatedAt, ExpiresAt: token.ExpiresAt,
		RevokedAt: token.RevokedAt, Used: usage.UsageCount > 0, UsageCount: usage.UsageCount,
		LastUsedAt: usage.LastUsedAt, EnrollmentID: usage.EnrollmentID, EnrollmentHost: usage.EnrollmentHost,
	}
	row.Status = tokenStatus(row, time.Now().UTC())
	return row, nil
}

func tokenStatus(row tokenRow, now time.Time) string {
	if row.RevokedAt != nil {
		return "revoked"
	}
	if now.After(row.ExpiresAt) {
		return "expired"
	}
	if row.Used {
		return "used"
	}
	return "active"
}

type tokenFilters struct {
	active, expired, revoked, used, unused bool
	host                                   string
}

func filterTokenRows(rows []tokenRow, f tokenFilters) []tokenRow {
	var out []tokenRow
	for _, row := range rows {
		if f.host != "" && !strings.EqualFold(row.Hostname, f.host) {
			continue
		}
		if f.active && row.Status != "active" {
			continue
		}
		if f.expired && row.Status != "expired" {
			continue
		}
		if f.revoked && row.Status != "revoked" {
			continue
		}
		if f.used && !row.Used {
			continue
		}
		if f.unused && row.Used {
			continue
		}
		out = append(out, row)
	}
	return out
}

func renderTokenTable(out io.Writer, rows []tokenRow, wide bool) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if wide {
		fmt.Fprintln(w, "ID\tHOST\tSTATUS\tCREATED\tEXPIRES\tUSED\tUSAGE\tLAST USED\tENROLLMENT")
	} else {
		fmt.Fprintln(w, "ID\tHOST\tSTATUS\tCREATED\tEXPIRES\tUSED\tLAST USED")
	}
	for _, row := range rows {
		lastUsed := "never"
		if row.LastUsedAt != nil {
			lastUsed = row.LastUsedAt.Format(time.RFC3339)
		}
		used := "no"
		if row.Used {
			used = "yes"
		}
		if wide {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n", row.ID, row.Hostname, row.Status, row.CreatedAt.Format(time.RFC3339), row.ExpiresAt.Format(time.RFC3339), used, row.UsageCount, lastUsed, row.EnrollmentID)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", row.ID, row.Hostname, row.Status, row.CreatedAt.Format(time.RFC3339), row.ExpiresAt.Format(time.RFC3339), used, lastUsed)
		}
	}
	_ = w.Flush()
}

func renderTokenMarkdown(out io.Writer, rows []tokenRow, wide bool) {
	if wide {
		fmt.Fprintln(out, "| ID | Host | Status | Created | Expires | Used | Usage | Last used | Enrollment |")
		fmt.Fprintln(out, "|---|---|---|---|---|---|---:|---|---|")
	} else {
		fmt.Fprintln(out, "| ID | Host | Status | Created | Expires | Used | Last used |")
		fmt.Fprintln(out, "|---|---|---|---|---|---|---|")
	}
	for _, row := range rows {
		lastUsed := "never"
		if row.LastUsedAt != nil {
			lastUsed = row.LastUsedAt.Format(time.RFC3339)
		}
		used := "no"
		if row.Used {
			used = "yes"
		}
		if wide {
			fmt.Fprintf(out, "| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%d` | `%s` | `%s` |\n", row.ID, row.Hostname, row.Status, row.CreatedAt.Format(time.RFC3339), row.ExpiresAt.Format(time.RFC3339), used, row.UsageCount, lastUsed, row.EnrollmentID)
		} else {
			fmt.Fprintf(out, "| `%s` | `%s` | `%s` | `%s` | `%s` | `%s` | `%s` |\n", row.ID, row.Hostname, row.Status, row.CreatedAt.Format(time.RFC3339), row.ExpiresAt.Format(time.RFC3339), used, lastUsed)
		}
	}
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

func resolveRootPassword(value, envName, file, defaultFile string, encrypt bool) (string, string, error) {
	if !encrypt {
		return "", "", nil
	}
	if value != "" {
		return value, "", nil
	}
	if envName != "" {
		pass := strings.TrimSpace(os.Getenv(envName))
		if pass == "" {
			return "", "", fmt.Errorf("environment variable %s is empty or unset", envName)
		}
		return pass, "", nil
	}
	return resolvePassword("", file, defaultFile)
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
