package config

import (
	"os"
	"strconv"
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
	Enabled               bool             `mapstructure:"enabled" yaml:"enabled"`
	ServiceName           string           `mapstructure:"service_name" yaml:"service_name"`
	ServiceVersion        string           `mapstructure:"service_version" yaml:"service_version"`
	DeploymentEnvironment string           `mapstructure:"deployment_environment" yaml:"deployment_environment"`
	OTLPEndpoint          string           `mapstructure:"otlp_endpoint" yaml:"otlp_endpoint"`
	OTLPProtocol          string           `mapstructure:"otlp_protocol" yaml:"otlp_protocol"`
	SamplingRatio         float64          `mapstructure:"sampling_ratio" yaml:"sampling_ratio"`
	Traces                SignalConfig     `mapstructure:"traces" yaml:"traces"`
	Metrics               SignalConfig     `mapstructure:"metrics" yaml:"metrics"`
	Logs                  SignalConfig     `mapstructure:"logs" yaml:"logs"`
	Exporter              ExporterConfig   `mapstructure:"exporter" yaml:"exporter"`
	Sampling              SamplingConfig   `mapstructure:"sampling" yaml:"sampling"`
	Prometheus            PrometheusConfig `mapstructure:"prometheus" yaml:"prometheus"`
}

type SignalConfig struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

type ExporterConfig struct {
	Protocol string `mapstructure:"protocol" yaml:"protocol"`
	Endpoint string `mapstructure:"endpoint" yaml:"endpoint"`
	Insecure bool   `mapstructure:"insecure" yaml:"insecure"`
}

type SamplingConfig struct {
	Ratio float64 `mapstructure:"ratio" yaml:"ratio"`
}

type PrometheusConfig struct {
	Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
	Path    string `mapstructure:"path" yaml:"path"`
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
			Traces:                SignalConfig{Enabled: true},
			Metrics:               SignalConfig{Enabled: true},
			Logs:                  SignalConfig{Enabled: true},
			Exporter:              ExporterConfig{Protocol: "grpc", Endpoint: "localhost:4317", Insecure: true},
			Sampling:              SamplingConfig{Ratio: 1},
			Prometheus:            PrometheusConfig{Enabled: true, Path: "/metrics"},
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
	normalizeTelemetry(&cfg)
	return cfg, err
}

func normalizeTelemetry(cfg *Config) {
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		cfg.Telemetry.Exporter.Endpoint = v
		cfg.Telemetry.OTLPEndpoint = v
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); v != "" {
		cfg.Telemetry.Exporter.Protocol = v
		cfg.Telemetry.OTLPProtocol = v
	}
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		cfg.Telemetry.ServiceName = v
	}
	if v := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); v != "" {
		for _, pair := range strings.Split(v, ",") {
			key, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
			if !ok {
				continue
			}
			switch strings.TrimSpace(key) {
			case "service.name":
				cfg.Telemetry.ServiceName = strings.TrimSpace(value)
			case "deployment.environment", "deployment.environment.name":
				cfg.Telemetry.DeploymentEnvironment = strings.TrimSpace(value)
			}
		}
	}
	if v := os.Getenv("OTEL_TRACES_SAMPLER"); v != "" {
		switch strings.ToLower(v) {
		case "always_on":
			cfg.Telemetry.Sampling.Ratio = 1
			cfg.Telemetry.SamplingRatio = 1
		case "always_off":
			cfg.Telemetry.Sampling.Ratio = 0
			cfg.Telemetry.SamplingRatio = 0
		}
	}
	if v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		if ratio, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Telemetry.Sampling.Ratio = ratio
			cfg.Telemetry.SamplingRatio = ratio
		}
	}
	if cfg.Telemetry.Exporter.Endpoint == "" {
		cfg.Telemetry.Exporter.Endpoint = cfg.Telemetry.OTLPEndpoint
	}
	if cfg.Telemetry.Exporter.Protocol == "" {
		cfg.Telemetry.Exporter.Protocol = cfg.Telemetry.OTLPProtocol
	}
	if cfg.Telemetry.Sampling.Ratio == 0 && cfg.Telemetry.SamplingRatio != 0 {
		cfg.Telemetry.Sampling.Ratio = cfg.Telemetry.SamplingRatio
	}
	if cfg.Telemetry.OTLPEndpoint == "" {
		cfg.Telemetry.OTLPEndpoint = cfg.Telemetry.Exporter.Endpoint
	}
	if cfg.Telemetry.OTLPProtocol == "" {
		cfg.Telemetry.OTLPProtocol = cfg.Telemetry.Exporter.Protocol
	}
	if cfg.Telemetry.SamplingRatio == 0 {
		cfg.Telemetry.SamplingRatio = cfg.Telemetry.Sampling.Ratio
	}
	if cfg.Telemetry.Prometheus.Path == "" {
		cfg.Telemetry.Prometheus.Path = "/metrics"
	}
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
		"telemetry.traces.enabled":              cfg.Telemetry.Traces.Enabled,
		"telemetry.metrics.enabled":             cfg.Telemetry.Metrics.Enabled,
		"telemetry.logs.enabled":                cfg.Telemetry.Logs.Enabled,
		"telemetry.exporter.endpoint":           cfg.Telemetry.Exporter.Endpoint,
		"telemetry.exporter.protocol":           cfg.Telemetry.Exporter.Protocol,
		"telemetry.exporter.insecure":           cfg.Telemetry.Exporter.Insecure,
		"telemetry.sampling.ratio":              cfg.Telemetry.Sampling.Ratio,
		"telemetry.prometheus.enabled":          cfg.Telemetry.Prometheus.Enabled,
		"telemetry.prometheus.path":             cfg.Telemetry.Prometheus.Path,
		"log.level":                             cfg.Log.Level,
		"otel.exporter.otlp.endpoint":           cfg.Telemetry.OTLPEndpoint,
		"otel.exporter.otlp.protocol":           cfg.Telemetry.OTLPProtocol,
		"otel.resource.attributes.service.name": cfg.Telemetry.ServiceName,
	}
}
