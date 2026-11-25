package validate

import "testing"

func TestNewPortalAccessSecret(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		s, err := NewPortalAccessSecret()
		if err != nil {
			t.Fatal(err)
		}
		if !ValidPortalAccessSecret(s) {
			t.Fatalf("invalid secret %q", s)
		}
		seen[s] = true
	}
	if len(seen) < 10 {
		t.Fatalf("expected variety, got %d unique", len(seen))
	}
}

func TestParsePortalAccessKey(t *testing.T) {
	sec, phone, ok := ParsePortalAccessKey("a7k2-6281234567890")
	if !ok || sec != "a7k2" || phone != "6281234567890" {
		t.Fatalf("got %q %q %v", sec, phone, ok)
	}
	if _, _, ok := ParsePortalAccessKey("A7K2-62812"); ok {
		t.Fatal("uppercase secret should fail")
	}
	if _, _, ok := ParsePortalAccessKey("a7k-62812"); ok {
		t.Fatal("short secret should fail")
	}
	if _, _, ok := ParsePortalAccessKey("a7k2-"); ok {
		t.Fatal("empty phone should fail")
	}
	if _, _, ok := ParsePortalAccessKey("a7k2-62812-extra"); ok {
		t.Fatal("extra dash should fail")
	}
}
