package rbac

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/config"
	"github.com/parisnakitakejser/ironroot/internal/db"
	"go.yaml.in/yaml/v3"
)

type Result struct {
	Files           []string
	Users           int
	Groups          int
	ServiceAccounts int
	RootCAs         int
	IntermediateCAs int
	Roles           int
	RoleBindings    int
	CARoles         int
	CARoleBindings  int
	TokenPolicies   int
}

type manifest struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name        string            `yaml:"name"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

type Spec struct {
	DisplayName      string    `yaml:"displayName"`
	Environment      string    `yaml:"environment"`
	Fingerprint      string    `yaml:"fingerprint"`
	Status           string    `yaml:"status"`
	TrustDomain      string    `yaml:"trustDomain"`
	RootRef          string    `yaml:"rootRef"`
	Owner            string    `yaml:"owner"`
	Namespace        string    `yaml:"namespace"`
	Subjects         []Subject `yaml:"subjects"`
	Rules            []Rule    `yaml:"rules"`
	RoleRef          RoleRef   `yaml:"roleRef"`
	IntermediateRef  string    `yaml:"intermediateRef"`
	CertificateTypes []string  `yaml:"certificateTypes"`
	AllowedDNS       []string  `yaml:"allowedDNS"`
	AllowedUsages    []string  `yaml:"allowedUsages"`
	MaxTTL           string    `yaml:"maxTTL"`
	IssuanceLimit    int       `yaml:"issuanceLimit"`
	RenewalAllowed   *bool     `yaml:"renewalAllowed"`
	RequireApproval  bool      `yaml:"requireApproval"`
	ExpiresAt        string    `yaml:"expiresAt"`
}

type Subject struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}

type Rule struct {
	Resources       []string `yaml:"resources"`
	Verbs           []string `yaml:"verbs"`
	IntermediateRef string   `yaml:"intermediateRef"`
}

type RoleRef struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}

func LoadAndApply(ctx context.Context, cfg config.RBACConfig, store db.Store) (Result, error) {
	var result Result
	if !cfg.Enabled {
		return result, nil
	}
	if cfg.Mode == "" {
		cfg.Mode = "file"
	}
	if cfg.Mode != "file" {
		return result, fmt.Errorf("rbac mode %q is not supported; use file", cfg.Mode)
	}
	files, err := expandPaths(cfg.Paths)
	if err != nil {
		return result, err
	}
	result.Files = files
	manifests, err := readManifests(files)
	if err != nil {
		return result, err
	}
	return apply(ctx, store, manifests, result)
}

func expandPaths(patterns []string) ([]string, error) {
	seen := map[string]struct{}{}
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid RBAC path pattern %q: %w", pattern, err)
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			seen[match] = struct{}{}
		}
	}
	files := make([]string, 0, len(seen))
	for file := range seen {
		files = append(files, file)
	}
	sort.Strings(files)
	return files, nil
}

func readManifests(files []string) ([]manifest, error) {
	var out []manifest
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read RBAC file %s: %w", file, err)
		}
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		doc := 0
		for {
			var m manifest
			err := dec.Decode(&m)
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("parse RBAC file %s document %d: %w", file, doc+1, err)
			}
			doc++
			if m.Kind == "" && m.Metadata.Name == "" {
				continue
			}
			if err := validateManifest(m, file, doc); err != nil {
				return nil, err
			}
			out = append(out, m)
		}
	}
	return out, nil
}

func validateManifest(m manifest, file string, doc int) error {
	where := fmt.Sprintf("RBAC file %s document %d", file, doc)
	if strings.TrimSpace(m.APIVersion) == "" {
		return fmt.Errorf("%s is missing apiVersion", where)
	}
	if strings.TrimSpace(m.Kind) == "" {
		return fmt.Errorf("%s is missing kind", where)
	}
	if strings.TrimSpace(m.Metadata.Name) == "" {
		return fmt.Errorf("%s is missing metadata.name", where)
	}
	switch m.Kind {
	case "User", "Group", "ServiceAccount":
		return nil
	case "RootCA":
		if strings.TrimSpace(m.Spec.Fingerprint) == "" {
			return fmt.Errorf("%s RootCA %q must define spec.fingerprint", where, m.Metadata.Name)
		}
	case "IntermediateCA":
		if strings.TrimSpace(m.Spec.RootRef) == "" {
			return fmt.Errorf("%s IntermediateCA %q must define spec.rootRef", where, m.Metadata.Name)
		}
		if strings.TrimSpace(m.Spec.Fingerprint) == "" {
			return fmt.Errorf("%s IntermediateCA %q must define spec.fingerprint", where, m.Metadata.Name)
		}
	case "Role", "CARole", "GlobalRole":
		if len(m.Spec.Rules) == 0 {
			return fmt.Errorf("%s role %q must define at least one rule", where, m.Metadata.Name)
		}
	case "RoleBinding", "CARoleBinding", "GlobalRoleBinding":
		if strings.TrimSpace(m.Spec.RoleRef.Name) == "" {
			return fmt.Errorf("%s binding %q must define spec.roleRef.name", where, m.Metadata.Name)
		}
		if len(m.Spec.Subjects) == 0 {
			return fmt.Errorf("%s binding %q must define at least one subject", where, m.Metadata.Name)
		}
	case "TokenPolicy":
		if strings.TrimSpace(m.Spec.IntermediateRef) == "" {
			return fmt.Errorf("%s token policy %q must define spec.intermediateRef", where, m.Metadata.Name)
		}
		if len(m.Spec.CertificateTypes) == 0 {
			return fmt.Errorf("%s token policy %q must define spec.certificateTypes", where, m.Metadata.Name)
		}
	default:
		return fmt.Errorf("%s has unsupported kind %q", where, m.Kind)
	}
	for _, subject := range m.Spec.Subjects {
		if strings.TrimSpace(subject.Kind) == "" || strings.TrimSpace(subject.Name) == "" {
			return fmt.Errorf("%s has subject with missing kind or name", where)
		}
	}
	return nil
}

func apply(ctx context.Context, store db.Store, manifests []manifest, result Result) (Result, error) {
	roles := map[string]manifest{}
	for _, m := range manifests {
		switch m.Kind {
		case "User":
			result.Users++
		case "Group":
			result.Groups++
		case "ServiceAccount":
			result.ServiceAccounts++
		case "RootCA":
			if err := applyRootCA(ctx, store, m); err != nil {
				return result, err
			}
			result.RootCAs++
		case "IntermediateCA":
			if err := applyIntermediateCA(ctx, store, m); err != nil {
				return result, err
			}
			result.IntermediateCAs++
		case "Role", "CARole", "GlobalRole":
			roles[m.Metadata.Name] = m
			result.Roles++
			if strings.HasPrefix(m.Kind, "CA") {
				result.CARoles++
			}
		}
	}
	for _, m := range manifests {
		switch m.Kind {
		case "RoleBinding", "CARoleBinding", "GlobalRoleBinding":
			role, ok := roles[m.Spec.RoleRef.Name]
			if !ok {
				return result, fmt.Errorf("RBAC binding %q references missing role %q", m.Metadata.Name, m.Spec.RoleRef.Name)
			}
			if err := applyRoleBinding(ctx, store, role, m); err != nil {
				return result, err
			}
			result.RoleBindings++
			if strings.HasPrefix(m.Kind, "CA") {
				result.CARoleBindings++
			}
		case "TokenPolicy":
			if err := applyTokenPolicy(ctx, store, m); err != nil {
				return result, err
			}
			result.TokenPolicies++
		}
	}
	return result, nil
}

func applyRootCA(ctx context.Context, store db.Store, m manifest) error {
	now := time.Now().UTC()
	status := firstNonEmpty(m.Spec.Status, "active")
	root := db.RootCA{
		ID:          m.Metadata.Name,
		Name:        firstNonEmpty(m.Spec.DisplayName, m.Metadata.Name),
		Environment: firstNonEmpty(m.Spec.Environment, "development"),
		Fingerprint: m.Spec.Fingerprint,
		Status:      status,
		TrustDomain: m.Spec.TrustDomain,
		CreatedAt:   now,
		NotBefore:   now,
		NotAfter:    now.Add(10 * 365 * 24 * time.Hour),
	}
	if err := store.UpsertRootCA(ctx, root); err != nil {
		return fmt.Errorf("apply RBAC RootCA %q: %w", m.Metadata.Name, err)
	}
	return nil
}

func applyIntermediateCA(ctx context.Context, store db.Store, m manifest) error {
	now := time.Now().UTC()
	maxTTL, err := parseDuration(m.Spec.MaxTTL)
	if err != nil {
		return fmt.Errorf("IntermediateCA %q has invalid maxTTL: %w", m.Metadata.Name, err)
	}
	intermediate := db.IntermediateCA{
		ID:              m.Metadata.Name,
		RootID:          m.Spec.RootRef,
		Name:            firstNonEmpty(m.Spec.DisplayName, m.Metadata.Name),
		Environment:     firstNonEmpty(m.Spec.Environment, "development"),
		Owner:           m.Spec.Owner,
		Namespace:       m.Spec.Namespace,
		Fingerprint:     m.Spec.Fingerprint,
		Status:          firstNonEmpty(m.Spec.Status, "active"),
		MaxTTL:          maxTTL,
		AllowedDNS:      strings.Join(m.Spec.AllowedDNS, ","),
		AllowedUsages:   strings.Join(m.Spec.AllowedUsages, ","),
		RequireApproval: m.Spec.RequireApproval,
		IssuanceLimit:   m.Spec.IssuanceLimit,
		RenewalAllowed:  true,
		CreatedAt:       now,
		NotBefore:       now,
		NotAfter:        now.Add(365 * 24 * time.Hour),
	}
	if m.Spec.RenewalAllowed != nil {
		intermediate.RenewalAllowed = *m.Spec.RenewalAllowed
	}
	if err := store.UpsertIntermediateCA(ctx, intermediate); err != nil {
		return fmt.Errorf("apply RBAC IntermediateCA %q: %w", m.Metadata.Name, err)
	}
	return nil
}

func applyRoleBinding(ctx context.Context, store db.Store, role, binding manifest) error {
	for _, rule := range role.Spec.Rules {
		intermediate := firstNonEmpty(rule.IntermediateRef, role.Spec.IntermediateRef, binding.Spec.IntermediateRef)
		if intermediate == "" {
			continue
		}
		for _, subject := range binding.Spec.Subjects {
			r := db.CARole{
				ID:             binding.Metadata.Name + "-" + strings.ToLower(subject.Kind) + "-" + subject.Name + "-" + intermediate,
				Name:           role.Metadata.Name,
				Subject:        strings.ToLower(subject.Kind) + ":" + subject.Name,
				IntermediateID: intermediate,
				Permissions:    strings.Join(rule.Verbs, ","),
				CreatedAt:      time.Now().UTC(),
			}
			if err := store.UpsertCARole(ctx, r); err != nil {
				return fmt.Errorf("apply RBAC role binding %q: %w", binding.Metadata.Name, err)
			}
		}
	}
	return nil
}

func applyTokenPolicy(ctx context.Context, store db.Store, m manifest) error {
	maxTTL, err := parseDuration(m.Spec.MaxTTL)
	if err != nil {
		return fmt.Errorf("token policy %q has invalid maxTTL: %w", m.Metadata.Name, err)
	}
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	if strings.TrimSpace(m.Spec.ExpiresAt) != "" {
		expiresAt, err = time.Parse(time.RFC3339, strings.TrimSpace(m.Spec.ExpiresAt))
		if err != nil {
			return fmt.Errorf("token policy %q has invalid expiresAt: %w", m.Metadata.Name, err)
		}
	}
	renewalAllowed := false
	if m.Spec.RenewalAllowed != nil {
		renewalAllowed = *m.Spec.RenewalAllowed
	}
	policy := db.CATokenPolicy{
		ID:               m.Metadata.Name,
		Name:             firstNonEmpty(m.Spec.DisplayName, m.Metadata.Name),
		IntermediateID:   m.Spec.IntermediateRef,
		CertificateTypes: strings.Join(m.Spec.CertificateTypes, ","),
		AllowedDNS:       strings.Join(m.Spec.AllowedDNS, ","),
		MaxTTL:           maxTTL,
		IssuanceLimit:    m.Spec.IssuanceLimit,
		RenewalAllowed:   renewalAllowed,
		RequireApproval:  m.Spec.RequireApproval,
		CreatedAt:        time.Now().UTC(),
		ExpiresAt:        expiresAt,
	}
	if err := store.UpsertCATokenPolicy(ctx, policy); err != nil {
		return fmt.Errorf("apply RBAC token policy %q: %w", m.Metadata.Name, err)
	}
	return nil
}

func parseDuration(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return time.ParseDuration(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
