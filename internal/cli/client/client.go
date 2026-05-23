package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/ironroot/ironroot/internal/ca"
	"github.com/ironroot/ironroot/internal/config"
	"github.com/ironroot/ironroot/internal/telemetry"
	apiclient "github.com/ironroot/ironroot/pkg/client"
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
	var server, token string
	cmd := tracedCommand("enroll", debug, func(ctx context.Context, cmd *cobra.Command, args []string) error {
		hostname, _ := os.Hostname()
		machineID := loadMachineID()
		resp, err := apiclient.New(server).Enroll(ctx, apiclient.EnrollmentRequest{Token: token, Hostname: hostname, MachineID: machineID, MAC: firstMAC()})
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	})
	cmd.Flags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.Flags().StringVar(&token, "token", "", "bootstrap token")
	_ = cmd.MarkFlagRequired("token")
	return cmd
}

func requestCert(debug bool) *cobra.Command {
	var server, dns, out, enrollmentID string
	cmd := tracedCommand("request-cert", debug, func(ctx context.Context, cmd *cobra.Command, args []string) error {
		names := splitCSV(dns)
		key, csr, err := ca.GenerateKeyAndCSR(names[0], names)
		if err != nil {
			return err
		}
		resp, err := apiclient.New(server).RequestCertificate(ctx, apiclient.CertificateRequest{EnrollmentID: enrollmentID, CSRPEM: string(csr), DNSNames: names})
		if err != nil {
			return err
		}
		return writeBundle(out, key, resp)
	})
	cmd.Flags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.Flags().StringVar(&dns, "dns", "", "comma-separated DNS names")
	cmd.Flags().StringVar(&out, "out", ".", "certificate output directory")
	cmd.Flags().StringVar(&enrollmentID, "enrollment-id", "", "enrollment id from enroll")
	_ = cmd.MarkFlagRequired("dns")
	_ = cmd.MarkFlagRequired("enrollment-id")
	return cmd
}

func renew(debug bool) *cobra.Command {
	var server, certPath, keyPath, out, enrollmentID, dns string
	cmd := tracedCommand("renew", debug, func(ctx context.Context, cmd *cobra.Command, args []string) error {
		_ = certPath
		_ = keyPath
		names := splitCSV(dns)
		key, csr, err := ca.GenerateKeyAndCSR(names[0], names)
		if err != nil {
			return err
		}
		resp, err := apiclient.New(server).RenewCertificate(ctx, apiclient.CertificateRequest{EnrollmentID: enrollmentID, CSRPEM: string(csr), DNSNames: names})
		if err != nil {
			return err
		}
		return writeBundle(out, key, resp)
	})
	cmd.Flags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.Flags().StringVar(&certPath, "cert", "", "existing certificate path")
	cmd.Flags().StringVar(&keyPath, "key", "", "existing key path")
	cmd.Flags().StringVar(&dns, "dns", "", "comma-separated DNS names")
	cmd.Flags().StringVar(&out, "out", ".", "certificate output directory")
	cmd.Flags().StringVar(&enrollmentID, "enrollment-id", "", "enrollment id")
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
