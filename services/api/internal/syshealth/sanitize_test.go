package syshealth

import "testing"

func TestSanitizeTextRedactsSecrets(t *testing.T) {
	in := `Bearer abcdef123456 Authorization cookie: session=xyz token=secretvalue email user@example.com`
	out := SanitizeText(in)
	if containsAny(out, "abcdef123456", "secretvalue", "user@example.com", "session=xyz") {
		t.Fatalf("secrets leaked: %s", out)
	}
}

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint("api", "auth", "boom", "failed for store 11111111-1111-1111-1111-111111111111", "main.go:12")
	b := Fingerprint("api", "auth", "boom", "failed for store 22222222-2222-2222-2222-222222222222", "main.go:99")
	if a != b {
		t.Fatalf("expected same fingerprint got %s vs %s", a, b)
	}
	c := Fingerprint("api", "auth", "other", "failed", "")
	if a == c {
		t.Fatal("different codes should differ")
	}
}

func TestSanitizeContext(t *testing.T) {
	m := SanitizeContext(map[string]any{
		"path":     "/api/v1/x",
		"password": "hunter2",
		"note":     "ok",
	})
	if m["password"] != "[REDACTED]" {
		t.Fatalf("password not redacted: %v", m["password"])
	}
	if m["note"] != "ok" {
		t.Fatalf("note changed")
	}
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if len(p) > 0 && contains(s, p) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(sub) > 0 && (s == sub || len(s) >= len(sub) && (indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
