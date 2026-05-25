package irtop

import "testing"

func TestViewLookupTablesCoverMainViewSpecs(t *testing.T) {
	for _, spec := range mainViewSpecs {
		if got := ParseView(spec.Name); got != spec.View {
			t.Fatalf("ParseView(%q) = %v, want %v", spec.Name, got, spec.View)
		}
		if got, ok := viewForKey(spec.Key); !ok || got != spec.View {
			t.Fatalf("viewForKey(%q) = %v, %t; want %v, true", spec.Key, got, ok, spec.View)
		}
		if got := viewName(spec.View); got != spec.Name {
			t.Fatalf("viewName(%v) = %q, want %q", spec.View, got, spec.Name)
		}
		for _, alias := range spec.Aliases {
			if got := ParseView(alias); got != spec.View {
				t.Fatalf("ParseView(%q) = %v, want %v", alias, got, spec.View)
			}
			if !validViewName(alias) {
				t.Fatalf("validViewName(%q) = false, want true", alias)
			}
		}
	}
}

func TestViewLookupFallbacks(t *testing.T) {
	if got := ParseView("missing"); got != ViewOverview {
		t.Fatalf("ParseView missing = %v, want overview", got)
	}
	if _, ok := viewForKey("0"); ok {
		t.Fatal("expected missing view key")
	}
	if got := viewName(ViewHelp); got != "help" {
		t.Fatalf("help view name = %q", got)
	}
}
