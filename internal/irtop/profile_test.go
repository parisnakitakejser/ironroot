package irtop

import "testing"

func TestProfileSetActiveIndexFallsBackToFirstProfile(t *testing.T) {
	profiles := ProfileSet{
		Profiles: []Profile{
			{Name: "local", Config: Config{Server: "http://localhost:8443"}},
			{Name: "prod", Config: Config{Server: "https://ironroot.example.com:8443"}},
		},
		Active: 99,
	}

	if got := profiles.ActiveIndex(); got != 0 {
		t.Fatalf("active index = %d, want 0", got)
	}
	if got := profiles.ActiveConfig().Server; got != "http://localhost:8443" {
		t.Fatalf("active server = %q", got)
	}
}

func TestProfileSetSetActiveConfigUsesSafeActiveIndex(t *testing.T) {
	profiles := ProfileSet{
		Profiles: []Profile{
			{Name: "local", Config: Config{Server: "http://localhost:8443"}},
		},
		Active: -1,
	}

	profiles.SetActiveConfig(Config{Server: "http://localhost:9443"})
	if got := profiles.Profiles[0].Config.Server; got != "http://localhost:9443" {
		t.Fatalf("active server = %q", got)
	}
}
