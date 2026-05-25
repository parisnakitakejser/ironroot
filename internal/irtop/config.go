package irtop

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

var ErrMissingDefaultConfig = errors.New("missing default irtop config")

type Config struct {
	Server             string        `yaml:"server"`
	Endpoint           string        `yaml:"endpoint"`
	Token              string        `yaml:"token"`
	CAFile             string        `yaml:"ca_file"`
	InsecureSkipVerify bool          `yaml:"insecure_skip_verify"`
	Refresh            time.Duration `yaml:"refresh"`
	DefaultView        string        `yaml:"default_view"`
	Output             string        `yaml:"output"`
}

type Profile struct {
	Name   string
	Config Config
}

type ProfileSet struct {
	Profiles []Profile
	Active   int
}

type configFile struct {
	DefaultProfile string            `yaml:"default_profile"`
	Profiles       map[string]Config `yaml:"profiles"`
}

func DefaultConfig() Config {
	return Config{Refresh: 5 * time.Second, DefaultView: "overview", Output: "tui"}
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for irtop config: %w", err)
	}
	return filepath.Join(home, ".ironroot", "config"), nil
}

func LoadConfig(path string) (Config, error) {
	profiles, err := LoadProfiles(path)
	if err != nil {
		return DefaultConfig(), err
	}
	return profiles.ActiveConfig(), nil
}

func LoadProfiles(path string) (ProfileSet, error) {
	usingDefault := path == ""
	if path == "" {
		defaultPath, err := DefaultConfigPath()
		if err != nil {
			return ProfileSet{}, err
		}
		path = defaultPath
	}
	path = expandPath(path)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			msg := fmt.Sprintf("irtop config file %s is missing; create ~/.ironroot/config or pass --config", path)
			if usingDefault {
				return ProfileSet{}, fmt.Errorf("%s: %w", msg, ErrMissingDefaultConfig)
			}
			return ProfileSet{}, errors.New(msg)
		}
		return ProfileSet{}, fmt.Errorf("read irtop config file %s: %w", path, err)
	}
	profiles, err := parseProfiles(b, path)
	if err != nil {
		return ProfileSet{}, err
	}
	return profiles, nil
}

func IsMissingDefaultConfig(err error) bool {
	return errors.Is(err, ErrMissingDefaultConfig)
}

func parseProfiles(data []byte, path string) (ProfileSet, error) {
	if err := validateProfileYAML(data, path); err != nil {
		return ProfileSet{}, err
	}
	var raw configFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		return ProfileSet{}, fmt.Errorf("parse irtop config file %s as YAML: %w", path, err)
	}
	if len(raw.Profiles) == 0 {
		return ProfileSet{}, fmt.Errorf("irtop config file %s is missing required value: profiles", path)
	}
	if err := requireProfileFields(data, path); err != nil {
		return ProfileSet{}, err
	}
	return buildProfileSet(raw, path)
}

func validateProfileYAML(data []byte, path string) error {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse irtop config file %s as YAML: %w", path, err)
	}
	if len(root.Content) == 0 {
		return nil
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return fmt.Errorf("irtop config file %s must contain a YAML mapping", path)
	}
	profilesNode := mappingValue(doc, "profiles")
	if profilesNode == nil {
		return nil
	}
	if profilesNode.Kind != yaml.MappingNode {
		return fmt.Errorf("irtop config file %s profiles must be a mapping of profile names to config values", path)
	}
	seen := map[string]struct{}{}
	for i := 0; i < len(profilesNode.Content); i += 2 {
		key := profilesNode.Content[i]
		value := profilesNode.Content[i+1]
		name := strings.TrimSpace(key.Value)
		if name == "" {
			return fmt.Errorf("irtop config file %s contains a profile with an empty name", path)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("irtop config file %s contains duplicate profile name %q", path, name)
		}
		seen[name] = struct{}{}
		if value.Kind != yaml.MappingNode {
			return fmt.Errorf("irtop config file %s profile %q must be a YAML mapping", path, name)
		}
	}
	return nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func buildProfileSet(raw configFile, path string) (ProfileSet, error) {
	names := make([]string, 0, len(raw.Profiles))
	for name := range raw.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	active := 0
	profiles := make([]Profile, 0, len(names))
	for i, name := range names {
		cfg := raw.Profiles[name]
		if strings.TrimSpace(name) == "" {
			return ProfileSet{}, fmt.Errorf("irtop config file %s contains a profile with an empty name", path)
		}
		if err := validateConfig(cfg, fmt.Sprintf("%s profile %q", path, name)); err != nil {
			return ProfileSet{}, err
		}
		cfg = normalizeConfig(cfg)
		cfg.CAFile = expandPath(cfg.CAFile)
		profiles = append(profiles, Profile{Name: name, Config: cfg})
		if raw.DefaultProfile == name {
			active = i
		}
	}
	if raw.DefaultProfile != "" {
		found := false
		for _, name := range names {
			if raw.DefaultProfile == name {
				found = true
				break
			}
		}
		if !found {
			return ProfileSet{}, fmt.Errorf("irtop config file %s default_profile %q does not match any profile", path, raw.DefaultProfile)
		}
	}
	return ProfileSet{Profiles: profiles, Active: active}, nil
}

func (p ProfileSet) ActiveConfig() Config {
	if len(p.Profiles) == 0 || p.Active < 0 || p.Active >= len(p.Profiles) {
		return DefaultConfig()
	}
	return p.Profiles[p.Active].Config
}

func (p *ProfileSet) Select(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for i, profile := range p.Profiles {
		if profile.Name == name {
			p.Active = i
			return nil
		}
	}
	return fmt.Errorf("irtop profile %q was not found in config", name)
}

func requireProfileFields(data []byte, path string) error {
	var fields struct {
		Profiles map[string]map[string]any `yaml:"profiles"`
	}
	if err := yaml.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("parse irtop config file %s as YAML: %w", path, err)
	}
	for name, profile := range fields.Profiles {
		for _, field := range []string{"server", "refresh", "default_view", "output"} {
			if field == "server" {
				if _, ok := profile["server"]; ok {
					continue
				}
				if _, ok := profile["endpoint"]; ok {
					continue
				}
				return fmt.Errorf("irtop config file %s profile %q is missing required value: server", path, name)
			}
			if _, ok := profile[field]; !ok {
				return fmt.Errorf("irtop config file %s profile %q is missing required value: %s", path, name, field)
			}
		}
	}
	return nil
}

func validateConfig(cfg Config, path string) error {
	if strings.TrimSpace(cfg.Server) != "" && strings.TrimSpace(cfg.Endpoint) != "" && strings.TrimSpace(cfg.Server) != strings.TrimSpace(cfg.Endpoint) {
		return fmt.Errorf("irtop config file %s defines both server and endpoint with different values", path)
	}
	if strings.TrimSpace(cfg.Server) == "" && strings.TrimSpace(cfg.Endpoint) == "" {
		return fmt.Errorf("irtop config file %s is missing required value: server", path)
	}
	if cfg.Refresh <= 0 {
		return fmt.Errorf("irtop config file %s has invalid refresh %q; use a positive duration such as 5s", path, cfg.Refresh)
	}
	if cfg.DefaultView == "" {
		return fmt.Errorf("irtop config file %s is missing required value: default_view", path)
	}
	if !validViewName(cfg.DefaultView) {
		return fmt.Errorf("irtop config file %s has invalid default_view %q", path, cfg.DefaultView)
	}
	if cfg.Output == "" {
		return fmt.Errorf("irtop config file %s is missing required value: output", path)
	}
	if cfg.Output != "tui" && cfg.Output != "text" {
		return fmt.Errorf("irtop config file %s has invalid output %q; use tui or text", path, cfg.Output)
	}
	return nil
}

func normalizeConfig(cfg Config) Config {
	if strings.TrimSpace(cfg.Server) == "" {
		cfg.Server = cfg.Endpoint
	}
	return cfg
}

func validViewName(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "overview", "certificates", "certs", "enrollments", "tokens", "ca-health", "security", "telemetry", "audit-log", "server":
		return true
	default:
		return false
	}
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
