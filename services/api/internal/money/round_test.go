package money

import "testing"

func TestRoundHalfUpIDR(t *testing.T) {
	cases := []struct {
		in   float64
		want int64
	}{
		{0, 0},
		{1.4, 1},
		{1.5, 2},
		{1.6, 2},
		{245000.4, 245000},
		{245000.5, 245001},
		{-1.5, -2},
	}
	for _, c := range cases {
		if got := RoundHalfUpIDR(c.in); got != c.want {
			t.Fatalf("RoundHalfUpIDR(%v)=%d want %d", c.in, got, c.want)
		}
	}
}

func TestPercentOfIDR(t *testing.T) {
	// 2.5% of 100_000 = 2500
	if got := PercentOfIDR(100_000, 2.5); got != 2500 {
		t.Fatalf("got %d", got)
	}
	// 10% of 15 = 1.5 → 2
	if got := PercentOfIDR(15, 10); got != 2 {
		t.Fatalf("got %d", got)
	}
}
