package irtop

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Server             string        `yaml:"server"`
	Token              string        `yaml:"token"`
	CAFile             string        `yaml:"ca_file"`
	InsecureSkipVerify bool          `yaml:"insecure_skip_verify"`
	Refresh            time.Duration `yaml:"refresh"`
	DefaultView        string        `yaml:"default_view"`
	Output             string        `yaml:"output"`
}

func DefaultConfig() Config {
	return Config{Server: "https://localhost:8443", Refresh: 5 * time.Second, DefaultView: "overview"}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	path = expandPath(path)
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Server == "" {
		cfg.Server = "https://localhost:8443"
	}
	if cfg.Refresh <= 0 {
		cfg.Refresh = 5 * time.Second
	}
	if cfg.DefaultView == "" {
		cfg.DefaultView = "overview"
	}
	cfg.CAFile = expandPath(cfg.CAFile)
	return cfg, nil
}

func expandPath(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
