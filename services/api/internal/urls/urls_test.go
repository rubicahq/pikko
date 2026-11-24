package urls

import "testing"

func TestGenerator(t *testing.T) {
	g := New("https://pikko.site", "https://app.pikko.site", "https://shop.pikko.site", "https://api.pikko.site")
	if got := g.Storefront("japanpicks"); got != "https://shop.pikko.site/japanpicks" {
		t.Fatalf("storefront: %s", got)
	}
	if got := g.StoreOverview("abc"); got != "https://app.pikko.site/stores/abc/overview" {
		t.Fatalf("overview: %s", got)
	}
	link := g.MagicLink("tok")
	if link != "https://app.pikko.site/auth/confirm?token=tok" {
		t.Fatalf("magic: %s", link)
	}
}

func TestGeneratorPikkoID(t *testing.T) {
	g := New("https://pikko.id", "https://app.pikko.id", "https://shop.pikko.id", "https://api.pikko.id")
	if got := g.Storefront("japanpicks"); got != "https://shop.pikko.id/japanpicks" {
		t.Fatalf("storefront: %s", got)
	}
	if got := g.MagicLink("tok"); got != "https://app.pikko.id/auth/confirm?token=tok" {
		t.Fatalf("magic: %s", got)
	}
	if got := g.Tracking("abc"); got != "https://shop.pikko.id/track/abc" {
		t.Fatalf("tracking: %s", got)
	}
	if got := g.Canonical("/pricing"); got != "https://pikko.id/pricing" {
		t.Fatalf("canonical: %s", got)
	}
}

func TestSafeRedirect(t *testing.T) {
	fb := "https://app.pikko.site/"
	got := SafeRedirect("https://evil.com/phish", []string{"pikko.site"}, fb)
	if got != fb {
		t.Fatalf("expected fallback, got %s", got)
	}
	got = SafeRedirect("https://app.pikko.site/stores", []string{"pikko.site"}, fb)
	if got != "https://app.pikko.site/stores" {
		t.Fatalf("got %s", got)
	}
	got = SafeRedirect("https://app.pikko.id/stores", []string{"pikko.id", "pikko.site"}, "https://app.pikko.id/")
	if got != "https://app.pikko.id/stores" {
		t.Fatalf("pikko.id allowlist: %s", got)
	}
}
