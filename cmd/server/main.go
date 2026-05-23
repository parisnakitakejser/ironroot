package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ironroot/ironroot/internal/api"
	"github.com/ironroot/ironroot/internal/audit"
	"github.com/ironroot/ironroot/internal/ca"
	"github.com/ironroot/ironroot/internal/config"
	"github.com/ironroot/ironroot/internal/db"
	"github.com/ironroot/ironroot/internal/telemetry"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.Getenv("IRONROOT_CONFIG"))
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := telemetry.NewLogger(os.Stdout, cfg.Log.Level)
	slog.SetDefault(logger)

	shutdownTelemetry, err := telemetry.Configure(ctx, cfg.Telemetry, "ironroot-server")
	if err != nil {
		logger.Error("configure telemetry", "error", err)
		os.Exit(1)
	}
	defer shutdownTelemetry(context.Background())

	store, err := db.Open(ctx, cfg.Database)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		logger.Error("run migrations", "error", err)
		os.Exit(1)
	}

	var authority ca.Authority
	authority, err = ca.LoadAuthority(cfg.PKI)
	if err != nil {
		logger.Warn("CA material unavailable; certificate issuance disabled until imported", "error", err)
		authority = ca.DisabledAuthority{}
	}

	srv := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           api.NewRouter(api.Dependencies{Config: cfg, Store: store, Authority: authority, Audit: audit.New(store), Logger: logger}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("starting ironroot-server", "addr", cfg.Server.Address)
		var err error
		if cfg.Server.TLS.CertFile != "" && cfg.Server.TLS.KeyFile != "" {
			err = srv.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown", "error", err)
	}
}
