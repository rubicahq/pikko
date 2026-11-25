package syshealth

import "testing"

func TestParseRangeDefaults(t *testing.T) {
	r, err := ParseRange("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Key != "24h" {
		t.Fatalf("want 24h got %s", r.Key)
	}
	if !r.To.After(r.From) {
		t.Fatal("invalid window")
	}
}

func TestParseRangeCustom(t *testing.T) {
	_, err := ParseRange("custom", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ParseRange("custom", "2026-01-02T00:00:00Z", "2026-01-01T00:00:00Z")
	if err == nil {
		t.Fatal("expected error")
	}
}
