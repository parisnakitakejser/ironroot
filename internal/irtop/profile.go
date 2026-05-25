package irtop

import (
	"fmt"
	"strings"
)

type Profile struct {
	Name   string
	Config Config
}

type ProfileSet struct {
	Profiles []Profile
	Active   int
}

func (p ProfileSet) Len() int {
	return len(p.Profiles)
}

func (p ProfileSet) HasMultiple() bool {
	return p.Len() > 1
}

func (p ProfileSet) ActiveIndex() int {
	if p.Active < 0 || p.Active >= len(p.Profiles) {
		return 0
	}
	return p.Active
}

func (p ProfileSet) ActiveConfig() Config {
	if len(p.Profiles) == 0 {
		return DefaultConfig()
	}
	return p.Profiles[p.ActiveIndex()].Config
}

func (p *ProfileSet) SetActiveConfig(cfg Config) {
	if len(p.Profiles) == 0 {
		return
	}
	p.Profiles[p.ActiveIndex()].Config = cfg
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
