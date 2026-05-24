package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/parisnakitakejser/ironroot/internal/config"
	"github.com/parisnakitakejser/ironroot/internal/irtop"
	"github.com/parisnakitakejser/ironroot/internal/telemetry"
)

func main() {
	var (
		configPath         string
		server             string
		caFile             string
		refresh            time.Duration
		insecureSkipVerify bool
		token              string
		output             string
	)

	flags := flag.NewFlagSet("irtop", flag.ExitOnError)
	flags.StringVar(&server, "server", "", "IronRoot API server URL")
	flags.StringVar(&configPath, "config", "", "irtop config file")
	flags.StringVar(&caFile, "ca-file", "", "CA bundle used to verify HTTPS connections")
	flags.DurationVar(&refresh, "refresh", 0, "dashboard refresh interval")
	flags.BoolVar(&insecureSkipVerify, "insecure-skip-verify", false, "skip TLS certificate verification")
	flags.StringVar(&token, "token", "", "read-only admin token")
	flags.StringVar(&output, "output", "", "output mode: tui or text")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "irtop - terminal monitoring UI for IronRoot\n\n")
		fmt.Fprintf(flags.Output(), "Usage:\n  irtop --server http://localhost:8443\n  irtop --server https://ironroot.example.com:8443 --ca-file ./root-ca.crt\n  irtop --server http://localhost:8443 --output text\n\n")
		fmt.Fprintf(flags.Output(), "Flags:\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cfg, err := irtop.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load irtop config: %v\n", err)
		os.Exit(2)
	}
	if server != "" {
		cfg.Server = server
	}
	if caFile != "" {
		cfg.CAFile = caFile
	}
	if refresh > 0 {
		cfg.Refresh = refresh
	}
	if insecureSkipVerify {
		cfg.InsecureSkipVerify = true
	}
	if token != "" {
		cfg.Token = token
	}
	if output != "" {
		cfg.Output = output
	}
	if cfg.Output == "" {
		cfg.Output = "tui"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	telemetryCfg := config.Default().Telemetry
	telemetryCfg.ServiceName = "irtop"
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		telemetryCfg.Enabled = true
		telemetryCfg.OTLPEndpoint = endpoint
		telemetryCfg.Exporter.Endpoint = endpoint
	}
	if protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); protocol != "" {
		telemetryCfg.OTLPProtocol = protocol
		telemetryCfg.Exporter.Protocol = protocol
	}
	if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
		telemetryCfg.ServiceName = serviceName
	}
	shutdown, err := telemetry.Configure(ctx, telemetryCfg, "irtop")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize telemetry: %v\n", err)
		os.Exit(2)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(shutdownCtx)
	}()

	if cfg.InsecureSkipVerify {
		fmt.Fprintln(os.Stderr, "Warning: TLS verification is disabled for this irtop session.")
	}

	client, err := irtop.NewClientChecked(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	switch cfg.Output {
	case "text":
		snapshot, err := client.Snapshot(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read IronRoot status: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(irtop.RenderText(snapshot))
	case "tui", "":
		program := tea.NewProgram(irtop.NewModel(client, cfg.Refresh, irtop.ParseView(cfg.DefaultView)))
		if _, err := program.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "irtop failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unsupported output mode %q. Use tui or text.\n", cfg.Output)
		os.Exit(2)
	}
}
