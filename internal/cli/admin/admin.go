package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/ironroot/ironroot/internal/config"
	ironcrypto "github.com/ironroot/ironroot/internal/crypto"
	"github.com/ironroot/ironroot/internal/db"
	"github.com/ironroot/ironroot/internal/telemetry"
	apiclient "github.com/ironroot/ironroot/pkg/client"
)

func New() *cobra.Command {
	var server string
	cmd := &cobra.Command{
		Use:   "ironroot-admin",
		Short: "Admin CLI for IronRoot PKI",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			_, _ = telemetry.Configure(cmd.Context(), config.Default().Telemetry, "ironroot-admin")
		},
	}
	cmd.PersistentFlags().StringVar(&server, "server", "http://localhost:8443", "IronRoot API URL")
	cmd.AddCommand(initServer(), importIntermediate(), createToken(), apiCommand("list-tokens", &server, func(ctx context.Context, c *apiclient.Client) error {
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

func initServer() *cobra.Command {
	return &cobra.Command{Use: "init-server", Short: "Initialize server configuration and migrations", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
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

func importIntermediate() *cobra.Command {
	return &cobra.Command{Use: "import-intermediate", Short: "Validate mounted intermediate CA material", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		_, err = os.Stat(cfg.PKI.IntermediateCertFile)
		return err
	}}
}

func createToken() *cobra.Command {
	var hostname, ttl string
	cmd := &cobra.Command{Use: "create-token", Short: "Create a hashed bootstrap token in the configured SQLite database", RunE: func(cmd *cobra.Command, args []string) error {
		if hostname == "" {
			return fmt.Errorf("--host is required")
		}
		dur, err := time.ParseDuration(ttl)
		if err != nil {
			return err
		}
		cfg, err := config.Load("")
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
