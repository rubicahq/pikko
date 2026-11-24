package validate

import "testing"

func TestValidateSlug(t *testing.T) {
	cases := []struct {
		slug string
		ok   bool
	}{
		{"japanpicks", true},
		{"ab", false},
		{"-abc", false},
		{"abc-", false},
		{"Admin", false},
		{"admin", false},
		{"my-store", true},
		{"my--store", false},
		{"store_1", false},
	}
	for _, c := range cases {
		if ValidateSlug(c.slug) != c.ok {
			t.Fatalf("ValidateSlug(%q)=%v want %v", c.slug, !c.ok, c.ok)
		}
	}
}

func TestSuggestSlug(t *testing.T) {
	s := SuggestSlug("Japan Picks!")
	if s != "japan-picks" {
		t.Fatalf("got %q", s)
	}
	s = SuggestSlug("Admin")
	if IsReservedSlug(s) {
		t.Fatalf("suggested reserved slug %q", s)
	}
}

func TestNormalizeEmail(t *testing.T) {
	_, n, ok := NormalizeEmail("  Ricky@Example.COM ")
	if !ok || n != "ricky@example.com" {
		t.Fatalf("got %q ok=%v", n, ok)
	}
	_, _, ok = NormalizeEmail("not-an-email")
	if ok {
		t.Fatal("expected invalid")
	}
}
