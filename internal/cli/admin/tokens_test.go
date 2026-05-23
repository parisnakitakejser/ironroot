package admin

import (
	"strings"
	"testing"
	"time"
)

func TestTokenStatus(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	if got := tokenStatus(tokenRow{ExpiresAt: now.Add(time.Hour)}, now); got != "active" {
		t.Fatalf("status = %s, want active", got)
	}
	if got := tokenStatus(tokenRow{ExpiresAt: now.Add(time.Hour), Used: true}, now); got != "used" {
		t.Fatalf("status = %s, want used", got)
	}
	if got := tokenStatus(tokenRow{ExpiresAt: now.Add(-time.Hour)}, now); got != "expired" {
		t.Fatalf("status = %s, want expired", got)
	}
	revokedAt := now.Add(-time.Minute)
	if got := tokenStatus(tokenRow{ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt}, now); got != "revoked" {
		t.Fatalf("status = %s, want revoked", got)
	}
}

func TestFilterTokenRows(t *testing.T) {
	rows := []tokenRow{
		{ID: "1", Hostname: "demo.local", Status: "active"},
		{ID: "2", Hostname: "demo.local", Status: "used", Used: true},
		{ID: "3", Hostname: "other.local", Status: "expired"},
	}
	filtered := filterTokenRows(rows, tokenFilters{host: "demo.local", unused: true})
	if len(filtered) != 1 || filtered[0].ID != "1" {
		t.Fatalf("unexpected filtered rows: %#v", filtered)
	}
	filtered = filterTokenRows(rows, tokenFilters{used: true})
	if len(filtered) != 1 || filtered[0].ID != "2" {
		t.Fatalf("unexpected used rows: %#v", filtered)
	}
}

func TestRenderTokenMarkdown(t *testing.T) {
	var b strings.Builder
	renderTokenMarkdown(&b, []tokenRow{{ID: "tok", Hostname: "demo.local", Status: "active", CreatedAt: time.Unix(0, 0).UTC(), ExpiresAt: time.Unix(60, 0).UTC()}}, false)
	if !strings.Contains(b.String(), "| `tok` | `demo.local` | `active` |") {
		t.Fatalf("unexpected markdown: %s", b.String())
	}
}
