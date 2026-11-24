package themes

import "testing"

func TestTwentyFourThemes(t *testing.T) {
	list := All()
	if len(list) != 24 {
		t.Fatalf("expected 24 themes, got %d", len(list))
	}
	seen := map[string]struct{}{}
	for _, th := range list {
		if !ValidHex(th.BackgroundColor) || !ValidHex(th.TextColor) || !ValidHex(th.PrimaryColor) {
			t.Fatalf("invalid hex on %s", th.ID)
		}
		if _, ok := seen[th.ID]; ok {
			t.Fatalf("duplicate theme id %s", th.ID)
		}
		seen[th.ID] = struct{}{}
		if AccessibleOn(th.PrimaryColor) == "" {
			t.Fatalf("missing button foreground for %s", th.ID)
		}
	}
}

func TestPRDHexValues(t *testing.T) {
	cases := map[string][3]string{
		"pikko_signature": {"#FFFFFF", "#15191E", "#FF3D00"},
		"bold_streetwear": {"#171717", "#FAFAFA", "#E7C94E"},
		"sakura":          {"#FFF3F6", "#583642", "#AD5573"},
		"contemporary":    {"#F4F4F7", "#272A35", "#5057A6"},
	}
	for id, want := range cases {
		th, ok := Get(id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if th.BackgroundColor != want[0] || th.TextColor != want[1] || th.PrimaryColor != want[2] {
			t.Fatalf("%s colors = %s/%s/%s want %v", id, th.BackgroundColor, th.TextColor, th.PrimaryColor, want)
		}
	}
}

func TestNormalizeHex(t *testing.T) {
	got, err := NormalizeHex(" #ff3d00 ")
	if err != nil || got != "#FF3D00" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := NormalizeHex("red"); err == nil {
		t.Fatal("expected error")
	}
}
