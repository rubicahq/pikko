package catalog

import "testing"

func TestNormalizeAttributeType(t *testing.T) {
	cases := map[string]string{
		"text":            AttrTypeShortText,
		"short_text":      AttrTypeShortText,
		"number":          AttrTypeNumber,
		"select":          AttrTypeSingleOption,
		"single_option":   AttrTypeSingleOption,
		"multiple_option": AttrTypeMultipleOption,
		"boolean":         AttrTypeShortText,
	}
	for in, want := range cases {
		got, ok := normalizeAttributeType(in)
		if !ok || got != want {
			t.Fatalf("%q → %q ok=%v, want %q", in, got, ok, want)
		}
	}
	if _, ok := normalizeAttributeType("nope"); ok {
		t.Fatal("expected invalid")
	}
}

func TestSanitizeAttributeKey(t *testing.T) {
	if got := sanitizeAttributeKey("Brand Name"); got != "brand_name" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeAttributeKey("123bad"); got != "attr_123bad" {
		t.Fatalf("got %q", got)
	}
	if !validAttributeKey(sanitizeAttributeKey("OK-Field")) {
		t.Fatal("expected valid key")
	}
}

func TestNormalizeOptionColor(t *testing.T) {
	if normalizeOptionColor("coral") != "coral" {
		t.Fatal("coral")
	}
	if normalizeOptionColor("neon") != "slate" {
		t.Fatal("fallback")
	}
	if len(AttributeOptionColors) != 10 {
		t.Fatalf("want 10 colors, got %d", len(AttributeOptionColors))
	}
}
