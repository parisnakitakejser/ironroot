package config

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server" yaml:"server"`
	Database  DatabaseConfig  `mapstructure:"database" yaml:"database"`
	PKI       PKIConfig       `mapstructure:"pki" yaml:"pki"`
	Telemetry TelemetryConfig `mapstructure:"telemetry" yaml:"telemetry"`
	Log       LogConfig       `mapstructure:"log" yaml:"log"`
}

type ServerConfig struct {
	Address string    `mapstructure:"address" yaml:"address"`
	TLS     TLSConfig `mapstructure:"tls" yaml:"tls"`
}

type TLSConfig struct {
	CertFile string `mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile  string `mapstructure:"key_file" yaml:"key_file"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver" yaml:"driver"`
	DSN    string `mapstructure:"dsn" yaml:"dsn"`
}

type PKIConfig struct {
	RootFile             string        `mapstructure:"root_file" yaml:"root_file"`
	ChainFile            string        `mapstructure:"chain_file" yaml:"chain_file"`
	IntermediateCertFile string        `mapstructure:"intermediate_cert_file" yaml:"intermediate_cert_file"`
	IntermediateKeyFile  string        `mapstructure:"intermediate_key_file" yaml:"intermediate_key_file"`
	IntermediateKeyPass  string        `mapstructure:"intermediate_key_pass" yaml:"intermediate_key_pass"`
	DefaultLifetime      time.Duration `mapstructure:"default_lifetime" yaml:"default_lifetime"`
	RenewBefore          time.Duration `mapstructure:"renew_before" yaml:"renew_before"`
}

type TelemetryConfig struct {
	Enabled               bool    `mapstructure:"enabled" yaml:"enabled"`
	ServiceName           string  `mapstructure:"service_name" yaml:"service_name"`
	ServiceVersion        string  `mapstructure:"service_version" yaml:"service_version"`
	DeploymentEnvironment string  `mapstructure:"deployment_environment" yaml:"deployment_environment"`
	OTLPEndpoint          string  `mapstructure:"otlp_endpoint" yaml:"otlp_endpoint"`
	OTLPProtocol          string  `mapstructure:"otlp_protocol" yaml:"otlp_protocol"`
	SamplingRatio         float64 `mapstructure:"sampling_ratio" yaml:"sampling_ratio"`
}

type LogConfig struct {
	Level string `mapstructure:"level" yaml:"level"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{Address: ":8443"},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "file:/data/ironroot.db?_foreign_keys=on",
		},
		PKI: PKIConfig{
			RootFile:             "/pki/root-ca.crt",
			ChainFile:            "/pki/ca-chain.crt",
			IntermediateCertFile: "/pki/intermediate.crt",
			IntermediateKeyFile:  "/pki/intermediate.key.enc",
			DefaultLifetime:      90 * 24 * time.Hour,
			RenewBefore:          30 * 24 * time.Hour,
		},
		Telemetry: TelemetryConfig{
			Enabled:               false,
			ServiceName:           "ironroot-server",
			ServiceVersion:        "0.1.0",
			DeploymentEnvironment: "dev",
			OTLPEndpoint:          "localhost:4317",
			OTLPProtocol:          "grpc",
			SamplingRatio:         1,
		},
		Log: LogConfig{Level: "info"},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("IRONROOT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if path == "" {
		path = "configs/server.yaml"
	}
	v.SetConfigFile(path)
	for k, val := range flatten(cfg, "") {
		v.SetDefault(k, val)
	}
	if _, err := os.Stat(path); err == nil {
		if err := v.ReadInConfig(); err != nil {
			return cfg, err
		}
	}
	err := v.Unmarshal(&cfg)
	return cfg, err
}

func flatten(cfg Config, prefix string) map[string]any {
	return map[string]any{
		"server.address":                        cfg.Server.Address,
		"server.tls.cert_file":                  cfg.Server.TLS.CertFile,
		"server.tls.key_file":                   cfg.Server.TLS.KeyFile,
		"database.driver":                       cfg.Database.Driver,
		"database.dsn":                          cfg.Database.DSN,
		"pki.root_file":                         cfg.PKI.RootFile,
		"pki.chain_file":                        cfg.PKI.ChainFile,
		"pki.intermediate_cert_file":            cfg.PKI.IntermediateCertFile,
		"pki.intermediate_key_file":             cfg.PKI.IntermediateKeyFile,
		"pki.intermediate_key_pass":             cfg.PKI.IntermediateKeyPass,
		"pki.default_lifetime":                  cfg.PKI.DefaultLifetime,
		"pki.renew_before":                      cfg.PKI.RenewBefore,
		"telemetry.enabled":                     cfg.Telemetry.Enabled,
		"telemetry.service_name":                cfg.Telemetry.ServiceName,
		"telemetry.service_version":             cfg.Telemetry.ServiceVersion,
		"telemetry.deployment_environment":      cfg.Telemetry.DeploymentEnvironment,
		"telemetry.otlp_endpoint":               cfg.Telemetry.OTLPEndpoint,
		"telemetry.otlp_protocol":               cfg.Telemetry.OTLPProtocol,
		"telemetry.sampling_ratio":              cfg.Telemetry.SamplingRatio,
		"log.level":                             cfg.Log.Level,
		"otel.exporter.otlp.endpoint":           cfg.Telemetry.OTLPEndpoint,
		"otel.exporter.otlp.protocol":           cfg.Telemetry.OTLPProtocol,
		"otel.resource.attributes.service.name": cfg.Telemetry.ServiceName,
	}
}
