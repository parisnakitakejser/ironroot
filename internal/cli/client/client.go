package client

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/parisnakitakejser/ironroot/internal/ca"
	"github.com/parisnakitakejser/ironroot/internal/config"
	"github.com/parisnakitakejser/ironroot/internal/telemetry"
	apiclient "github.com/parisnakitakejser/ironroot/pkg/client"
)

func New() *cobra.Command {
	var debug bool
	cmd := &cobra.Command{
		Use:   "ironroot-client",
		Short: "Client CLI for IronRoot enrollment and certificate lifecycle",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Default()
			cfg.Telemetry.ServiceName = "ironroot-client"
			_, err := telemetry.Configure(cmd.Context(), cfg.Telemetry, "ironroot-client")
			return err
		},
	}
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "print debug output including trace id")
	cmd.AddCommand(trust(debug), enroll(debug), requestCert(debug), renew(debug), status(debug))
	return cmd
}

func trust(debug bool) *cobra.Command {
	var server, out string
	cmd := &cobra.Command{Use: "trust", Short: "Manage local trust anchors"}
	install := tracedCommand("install", debug, func(ctx context.Context, cmd *cobra.Command, args []string) error {
		root, err := apiclient.New(server).Root(ctx)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(out, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(out, "root-ca.crt"), []byte(root), 0o644)
	})
	install.Flags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	install.Flags().StringVar(&out, "out", ".", "trust output directory")
	cmd.AddCommand(install)
	return cmd
}

func enroll(debug bool) *cobra.Command {
	var server, token, hostnameOverride string
	cmd := tracedCommand("enroll", debug, func(ctx context.Context, cmd *cobra.Command, args []string) error {
		hostname, _ := os.Hostname()
		if strings.TrimSpace(hostnameOverride) != "" {
			hostname = strings.TrimSpace(hostnameOverride)
		}
		machineID := loadMachineID()
		resp, err := apiclient.New(server).Enroll(ctx, apiclient.EnrollmentRequest{Token: token, Hostname: hostname, MachineID: machineID, MAC: firstMAC()})
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	})
	cmd.Flags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.Flags().StringVar(&token, "token", "", "bootstrap token")
	cmd.Flags().StringVar(&hostnameOverride, "hostname", "", "hostname to enroll; defaults to this machine's OS hostname")
	_ = cmd.MarkFlagRequired("token")
	return cmd
}

func requestCert(debug bool) *cobra.Command {
	var server, dns, out, enrollmentID string
	var overwrite, bundle bool
	var keyType, curve, format, keyPassword, keyPasswordEnv, chmodValue, owner, group string
	var rsaBits int
	cmd := tracedCommand("request-cert", debug, func(ctx context.Context, cmd *cobra.Command, args []string) error {
		if err := validateEnrollmentID(enrollmentID); err != nil {
			return err
		}
		names := splitCSV(dns)
		key, csr, keyInfo, err := generateKeyAndCSR(names[0], names, keyType, curve, rsaBits, keyPasswordFrom(keyPassword, keyPasswordEnv))
		if err != nil {
			return err
		}
		resp, err := apiclient.New(server).RequestCertificate(ctx, apiclient.CertificateRequest{EnrollmentID: enrollmentID, CSRPEM: string(csr), DNSNames: names})
		if err != nil {
			return err
		}
		target := out
		if strings.TrimSpace(target) == "" {
			target = defaultCertOutputDir(names[0])
		}
		if err := writeCertificateOutput(target, key, resp, certificateOutputOptions{
			DNSNames: names, EnrollmentID: enrollmentID, Overwrite: overwrite, Bundle: bundle, Format: format,
			Chmod: chmodValue, Owner: owner, Group: group, KeyInfo: keyInfo,
		}); err != nil {
			return err
		}
		printCertificateSuccess(cmd, target, resp, bundle)
		return nil
	})
	cmd.Flags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.Flags().StringVar(&dns, "dns", "", "comma-separated DNS names")
	cmd.Flags().StringVar(&out, "out", "", "certificate output directory; defaults to the platform IronRoot certificate directory")
	cmd.Flags().StringVar(&enrollmentID, "enrollment-id", "", "enrollment id from enroll")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "replace existing certificate files in the output directory")
	cmd.Flags().StringVar(&keyType, "key-type", "ecdsa", "private key type: ecdsa or rsa")
	cmd.Flags().IntVar(&rsaBits, "rsa-bits", 2048, "RSA key size when --key-type rsa")
	cmd.Flags().StringVar(&curve, "curve", "p256", "ECDSA curve when --key-type ecdsa: p256, p384, or p521")
	cmd.Flags().StringVar(&format, "format", "pem", "output format; currently pem")
	cmd.Flags().BoolVar(&bundle, "bundle", true, "generate fullchain.crt")
	cmd.Flags().StringVar(&keyPassword, "key-password", "", "encrypt tls.key with this password; avoid shell history in production")
	cmd.Flags().StringVar(&keyPasswordEnv, "key-password-env", "", "environment variable containing tls.key encryption password")
	cmd.Flags().StringVar(&chmodValue, "chmod", "", "file mode override such as 0600; defaults to 0600 for tls.key and 0644 for certificate files")
	cmd.Flags().StringVar(&owner, "owner", "", "optional file owner name or uid")
	cmd.Flags().StringVar(&group, "group", "", "optional file group name or gid")
	_ = cmd.MarkFlagRequired("dns")
	_ = cmd.MarkFlagRequired("enrollment-id")
	return cmd
}

func renew(debug bool) *cobra.Command {
	var server, certPath, keyPath, out, enrollmentID, dns string
	cmd := tracedCommand("renew", debug, func(ctx context.Context, cmd *cobra.Command, args []string) error {
		_ = certPath
		_ = keyPath
		if err := validateEnrollmentID(enrollmentID); err != nil {
			return err
		}
		names := splitCSV(dns)
		key, csr, err := ca.GenerateKeyAndCSR(names[0], names)
		if err != nil {
			return err
		}
		resp, err := apiclient.New(server).RenewCertificate(ctx, apiclient.CertificateRequest{EnrollmentID: enrollmentID, CSRPEM: string(csr), DNSNames: names})
		if err != nil {
			return err
		}
		if out == "" {
			out = defaultCertOutputDir(names[0])
		}
		return writeBundle(out, key, resp)
	})
	cmd.Flags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.Flags().StringVar(&certPath, "cert", "", "existing certificate path")
	cmd.Flags().StringVar(&keyPath, "key", "", "existing key path")
	cmd.Flags().StringVar(&dns, "dns", "", "comma-separated DNS names")
	cmd.Flags().StringVar(&out, "out", ".", "certificate output directory")
	cmd.Flags().StringVar(&enrollmentID, "enrollment-id", "", "enrollment id")
	_ = cmd.MarkFlagRequired("enrollment-id")
	_ = cmd.MarkFlagRequired("dns")
	return cmd
}

func status(debug bool) *cobra.Command {
	var server, serial string
	cmd := tracedCommand("status", debug, func(ctx context.Context, cmd *cobra.Command, args []string) error {
		resp, err := apiclient.New(server).Status(ctx, serial)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	})
	cmd.Flags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.Flags().StringVar(&serial, "serial", "", "certificate serial")
	_ = cmd.MarkFlagRequired("serial")
	return cmd
}

func tracedCommand(name string, debug bool, run func(context.Context, *cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{Use: name, RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		ctx, span := otel.Tracer("ironroot-client").Start(cmd.Context(), "pki-client "+name)
		defer span.End()
		err := run(ctx, cmd, args)
		telemetry.RecordCommand(ctx, name, start, err)
		if debug {
			fmt.Fprintf(os.Stderr, "trace_id=%s\n", trace.SpanContextFromContext(ctx).TraceID().String())
		}
		return err
	}}
}

func writeBundle(out string, key []byte, resp apiclient.CertificateResponse) error {
	if err := os.MkdirAll(out, 0o700); err != nil {
		return err
	}
	files := map[string][]byte{
		"tls.key":      key,
		"tls.crt":      []byte(resp.CertPEM),
		"ca-chain.crt": []byte(resp.ChainPEM),
		"root-ca.crt":  []byte(resp.RootPEM),
	}
	for name, body := range files {
		mode := os.FileMode(0o644)
		if name == "tls.key" {
			mode = 0o600
		}
		if err := os.WriteFile(filepath.Join(out, name), body, mode); err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

type keyMetadata struct {
	Type      string `json:"type"`
	Curve     string `json:"curve,omitempty"`
	RSAEnBits int    `json:"rsa_bits,omitempty"`
	Encrypted bool   `json:"encrypted"`
}

type certificateOutputOptions struct {
	DNSNames     []string
	EnrollmentID string
	Overwrite    bool
	Bundle       bool
	Format       string
	Chmod        string
	Owner        string
	Group        string
	KeyInfo      keyMetadata
}

func generateKeyAndCSR(commonName string, dnsNames []string, keyType, curve string, rsaBits int, password string) ([]byte, []byte, keyMetadata, error) {
	if strings.ToLower(strings.TrimSpace(keyType)) == "" {
		keyType = "ecdsa"
	}
	var signer crypto.Signer
	var err error
	meta := keyMetadata{Type: strings.ToLower(keyType), Encrypted: password != ""}
	switch meta.Type {
	case "ecdsa":
		c := elliptic.P256()
		meta.Curve = "p256"
		switch strings.ToLower(curve) {
		case "p256", "p-256", "":
			c = elliptic.P256()
			meta.Curve = "p256"
		case "p384", "p-384":
			c = elliptic.P384()
			meta.Curve = "p384"
		case "p521", "p-521":
			c = elliptic.P521()
			meta.Curve = "p521"
		default:
			return nil, nil, meta, fmt.Errorf("unsupported ECDSA curve %q; use p256, p384, or p521", curve)
		}
		signer, err = ecdsa.GenerateKey(c, rand.Reader)
	case "rsa":
		if rsaBits < 2048 {
			return nil, nil, meta, fmt.Errorf("RSA keys must be at least 2048 bits")
		}
		meta.RSAEnBits = rsaBits
		return generateRSAKeyAndCSR(commonName, dnsNames, rsaBits, password, meta)
	default:
		return nil, nil, meta, fmt.Errorf("unsupported key type %q; use ecdsa or rsa", keyType)
	}
	if err != nil {
		return nil, nil, meta, err
	}
	keyPEM, err := marshalPrivateKeyPEM(signer, password)
	if err != nil {
		return nil, nil, meta, err
	}
	csrPEM, err := createCSR(commonName, dnsNames, signer)
	if err != nil {
		return nil, nil, meta, err
	}
	return keyPEM, csrPEM, meta, nil
}

func generateRSAKeyAndCSR(commonName string, dnsNames []string, bits int, password string, meta keyMetadata) ([]byte, []byte, keyMetadata, error) {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, meta, err
	}
	keyPEM, err := marshalPrivateKeyPEM(key, password)
	if err != nil {
		return nil, nil, meta, err
	}
	csrPEM, err := createCSR(commonName, dnsNames, key)
	if err != nil {
		return nil, nil, meta, err
	}
	return keyPEM, csrPEM, meta, nil
}

func marshalPrivateKeyPEM(signer crypto.Signer, password string) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(signer)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	if password != "" {
		block, err = x509.EncryptPEMBlock(rand.Reader, "PRIVATE KEY", der, []byte(password), x509.PEMCipherAES256)
		if err != nil {
			return nil, err
		}
	}
	return pem.EncodeToMemory(block), nil
}

func createCSR(commonName string, dnsNames []string, signer crypto.Signer) ([]byte, error) {
	tpl := &x509.CertificateRequest{Subject: pkix.Name{CommonName: commonName}, DNSNames: dnsNames}
	csr, err := x509.CreateCertificateRequest(rand.Reader, tpl, signer)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr}), nil
}

func keyPasswordFrom(value, envName string) string {
	if value != "" {
		return value
	}
	if envName != "" {
		return strings.TrimSpace(os.Getenv(envName))
	}
	return ""
}

func defaultCertOutputDir(dnsName string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", "certs", safePathName(dnsName))
	}
	return defaultCertOutputDirForOS(runtime.GOOS, home, dnsName)
}

func defaultCertOutputDirForOS(goos, home, dnsName string) string {
	if goos == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "ironroot", "certs", safePathName(dnsName))
	}
	return filepath.Join(home, ".local", "share", "ironroot", "certs", safePathName(dnsName))
}

func safePathName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "localhost"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "\x00", "_")
	return replacer.Replace(value)
}

func writeCertificateOutput(out string, key []byte, resp apiclient.CertificateResponse, opts certificateOutputOptions) error {
	if strings.ToLower(opts.Format) != "pem" && opts.Format != "" {
		return fmt.Errorf("unsupported --format %q; only pem is currently supported", opts.Format)
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", out, err)
	}
	if err := validateOutputWritable(out); err != nil {
		return err
	}
	files, err := certificateFiles(key, resp, opts)
	if err != nil {
		return err
	}
	for name := range files {
		path := filepath.Join(out, name)
		if !opts.Overwrite {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("refusing to overwrite existing file %s; pass --overwrite to replace generated certificate files", path)
			}
		}
	}
	keyMode, certMode, err := outputModes(opts.Chmod)
	if err != nil {
		return err
	}
	uid, gid, chown, err := resolveOwnerGroup(opts.Owner, opts.Group)
	if err != nil {
		return err
	}
	for name, body := range files {
		mode := certMode
		if name == "tls.key" {
			mode = keyMode
		}
		path := filepath.Join(out, name)
		if err := os.WriteFile(path, body, mode); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		if chown {
			if err := os.Chown(path, uid, gid); err != nil {
				return fmt.Errorf("failed to set owner/group on %s: %w", path, err)
			}
		}
	}
	return nil
}

func validateOutputWritable(out string) error {
	testPath := filepath.Join(out, ".ironroot-write-test")
	if err := os.WriteFile(testPath, []byte("ok"), 0o600); err != nil {
		return fmt.Errorf("failed to access output directory %s: %w", out, err)
	}
	_ = os.Remove(testPath)
	return nil
}

func certificateFiles(key []byte, resp apiclient.CertificateResponse, opts certificateOutputOptions) (map[string][]byte, error) {
	fullchain := append([]byte(resp.CertPEM), []byte(resp.ChainPEM)...)
	files := map[string][]byte{
		"tls.key":          key,
		"tls.crt":          []byte(resp.CertPEM),
		"ca-chain.crt":     []byte(resp.ChainPEM),
		"metadata.json":    nil,
		"fingerprints.txt": nil,
		"README.txt":       nil,
	}
	if opts.Bundle {
		files["fullchain.crt"] = fullchain
	}
	metadata, err := certificateMetadata(resp, opts)
	if err != nil {
		return nil, err
	}
	files["metadata.json"] = metadata
	files["fingerprints.txt"] = []byte(certificateFingerprints(resp))
	files["README.txt"] = []byte(certificateReadme(opts.Bundle))
	return files, nil
}

func certificateMetadata(resp apiclient.CertificateResponse, opts certificateOutputOptions) ([]byte, error) {
	cert, err := parsePEMCertificate(resp.CertPEM)
	if err != nil {
		return nil, err
	}
	issuer := ""
	if chainCert, err := firstCertificateFromPEM(resp.ChainPEM); err == nil {
		issuer = chainCert.Subject.String()
	}
	body, err := json.MarshalIndent(map[string]any{
		"dns_names":      opts.DNSNames,
		"enrollment_id":  opts.EnrollmentID,
		"issuer":         issuer,
		"issued_at":      time.Now().UTC().Format(time.RFC3339),
		"not_before":     resp.NotBefore,
		"not_after":      resp.NotAfter,
		"algorithm":      cert.PublicKeyAlgorithm.String(),
		"key":            opts.KeyInfo,
		"fingerprint":    pemFingerprint(cert.Raw),
		"serial":         resp.Serial,
		"renew_before":   resp.RenewBefore,
		"bundle_written": opts.Bundle,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func certificateFingerprints(resp apiclient.CertificateResponse) string {
	cert, err := parsePEMCertificate(resp.CertPEM)
	if err != nil {
		return "tls.crt SHA256: unavailable\n"
	}
	return "tls.crt SHA256: " + pemFingerprint(cert.Raw) + "\n"
}

func certificateReadme(bundle bool) string {
	fullchainLine := ""
	if bundle {
		fullchainLine = "- fullchain.crt -> tls.crt plus CA chain, useful for nginx and Caddy\n"
	}
	return `This directory contains a certificate issued by IronRoot.

Files:
- tls.key       -> private key. Never share this file.
- tls.crt       -> issued server certificate.
- ca-chain.crt  -> Intermediate and Root CA chain.
` + fullchainLine + `- metadata.json -> issuance metadata and expiration details.
- fingerprints.txt -> SHA-256 fingerprints for debugging.

Nginx example:
ssl_certificate fullchain.crt;
ssl_certificate_key tls.key;

Caddy example:
tls tls.crt tls.key
`
}

func outputModes(chmodValue string) (os.FileMode, os.FileMode, error) {
	if strings.TrimSpace(chmodValue) == "" {
		return 0o600, 0o644, nil
	}
	parsed, err := strconv.ParseUint(chmodValue, 8, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid --chmod value %q; use an octal mode like 0600", chmodValue)
	}
	return os.FileMode(parsed), os.FileMode(parsed), nil
}

func resolveOwnerGroup(ownerName, groupName string) (int, int, bool, error) {
	if ownerName == "" && groupName == "" {
		return -1, -1, false, nil
	}
	uid, gid := -1, -1
	if ownerName != "" {
		u, err := user.Lookup(ownerName)
		if err != nil {
			if _, parseErr := strconv.Atoi(ownerName); parseErr != nil {
				return -1, -1, false, fmt.Errorf("unknown owner %q", ownerName)
			}
			uid, _ = strconv.Atoi(ownerName)
		} else {
			uid, _ = strconv.Atoi(u.Uid)
		}
	}
	if groupName != "" {
		g, err := user.LookupGroup(groupName)
		if err != nil {
			if _, parseErr := strconv.Atoi(groupName); parseErr != nil {
				return -1, -1, false, fmt.Errorf("unknown group %q", groupName)
			}
			gid, _ = strconv.Atoi(groupName)
		} else {
			gid, _ = strconv.Atoi(g.Gid)
		}
	}
	return uid, gid, true, nil
}

func parsePEMCertificate(value string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("response did not contain a certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

func firstCertificateFromPEM(value string) (*x509.Certificate, error) {
	for rest := []byte(value); len(rest) > 0; {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining
		if block.Type == "CERTIFICATE" {
			return x509.ParseCertificate(block.Bytes)
		}
	}
	return nil, fmt.Errorf("no certificate found")
}

func pemFingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	encoded := strings.ToUpper(hex.EncodeToString(sum[:]))
	parts := make([]string, 0, len(encoded)/2)
	for i := 0; i < len(encoded); i += 2 {
		parts = append(parts, encoded[i:i+2])
	}
	return strings.Join(parts, ":")
}

func printCertificateSuccess(cmd *cobra.Command, out string, resp apiclient.CertificateResponse, bundle bool) {
	fmt.Fprintln(cmd.OutOrStdout(), "Certificate generated successfully.")
	fmt.Fprintf(cmd.OutOrStdout(), "\nFiles written to:\n%s\n\n", out)
	fmt.Fprintln(cmd.OutOrStdout(), "Generated:")
	fmt.Fprintln(cmd.OutOrStdout(), "- tls.key")
	fmt.Fprintln(cmd.OutOrStdout(), "- tls.crt")
	if bundle {
		fmt.Fprintln(cmd.OutOrStdout(), "- fullchain.crt")
	}
	fmt.Fprintln(cmd.OutOrStdout(), "- ca-chain.crt")
	fmt.Fprintln(cmd.OutOrStdout(), "- metadata.json")
	fmt.Fprintln(cmd.OutOrStdout(), "- fingerprints.txt")
	fmt.Fprintln(cmd.OutOrStdout(), "- README.txt")
	fmt.Fprintf(cmd.OutOrStdout(), "\nSerial: %s\nExpires: %s\n\n", resp.Serial, resp.NotAfter)
	fmt.Fprintln(cmd.OutOrStdout(), "Next steps:")
	fmt.Fprintln(cmd.OutOrStdout(), "- Install the Root CA trust bundle if this is a local or private trust domain.")
	fmt.Fprintln(cmd.OutOrStdout(), "- Configure nginx, Caddy, Podman, Kubernetes, or your service with tls.crt/fullchain.crt and tls.key.")
	fmt.Fprintln(cmd.OutOrStdout(), "- Test HTTPS with the requested DNS name.")
}

func validateEnrollmentID(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--enrollment-id is required")
	}
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf("--enrollment-id must be the enrollment_id returned by `ironroot-client enroll`, not the bootstrap token")
	}
	return nil
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		out = []string{"localhost"}
	}
	return out
}

func loadMachineID() string {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(b)) != "" {
			return strings.TrimSpace(string(b))
		}
	}
	hostname, _ := os.Hostname()
	return hostname
}

func firstMAC() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if len(iface.HardwareAddr) > 0 {
			return iface.HardwareAddr.String()
		}
	}
	return ""
}
