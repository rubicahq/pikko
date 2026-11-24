package flipbook

import "testing"

func TestNormalizeBBox(t *testing.T) {
	in := HotspotInput{XPct: -2, YPct: 90, WidthPct: 20, HeightPct: 20}
	if !NormalizeBBox(&in) {
		t.Fatal("expected ok")
	}
	if in.XPct != 0 {
		t.Fatalf("x=%v", in.XPct)
	}
	if in.YPct+in.HeightPct > 100.0001 {
		t.Fatalf("y+h overflow %v+%v", in.YPct, in.HeightPct)
	}

	in2 := HotspotInput{XPct: 10, YPct: 10, WidthPct: 0, HeightPct: 0}
	if !NormalizeBBox(&in2) {
		t.Fatal("zero size should default to 10x10")
	}
	if in2.WidthPct != 10 || in2.HeightPct != 10 {
		t.Fatalf("defaults=%v %v", in2.WidthPct, in2.HeightPct)
	}
}
