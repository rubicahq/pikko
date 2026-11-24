package money

import "testing"

func TestValidateIDR(t *testing.T) {
	if err := ValidateIDR(1000, false); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIDR(-1, true); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateIDR(0, false); err == nil {
		t.Fatal("expected error for zero")
	}
}

func TestFormatIDR(t *testing.T) {
	if got := FormatIDR(245000); got != "Rp245.000" {
		t.Fatalf("got %s", got)
	}
	if got := FormatIDR(0); got != "Rp0" {
		t.Fatalf("got %s", got)
	}
}
