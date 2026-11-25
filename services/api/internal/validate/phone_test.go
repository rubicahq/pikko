package validate

import "testing"

func TestNormalizeIndonesianPhone(t *testing.T) {
	okCases := map[string]string{
		"0812-3456-7890":    "6281234567890",
		"+62 812 3456 7890": "6281234567890",
		"81234567890":       "6281234567890",
		"6281234567890":     "6281234567890",
	}
	for in, want := range okCases {
		got, ok := NormalizeIndonesianPhone(in)
		if !ok || got != want {
			t.Errorf("%q => (%q,%v) want (%q,true)", in, got, ok, want)
		}
	}
	bad := []string{"", "123", "0211234567", "+1 415 555 0100", "62812"}
	for _, in := range bad {
		if got, ok := NormalizeIndonesianPhone(in); ok {
			t.Errorf("%q unexpectedly ok => %q", in, got)
		}
	}
}
