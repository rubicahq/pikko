package whatsapp

import (
	"strings"
	"testing"
)

func TestRenderAndDeepLink(t *testing.T) {
	body := Render("Halo {{customer_name}}, order {{order_number}}", map[string]string{
		"customer_name": "Ani",
		"order_number":  "ORD-1",
	})
	if body != "Halo Ani, order ORD-1" {
		t.Fatalf("body=%q", body)
	}
	link := DeepLink("081234567890", body)
	if !strings.HasPrefix(link, "https://wa.me/6281234567890?text=") {
		t.Fatalf("link=%q", link)
	}
	if !strings.Contains(link, "Ani") || !strings.Contains(link, "ORD-1") {
		t.Fatalf("link missing payload %q", link)
	}
}

func TestNormalizePhoneDigits(t *testing.T) {
	cases := map[string]string{
		"0812-3456-7890": "6281234567890",
		"+62 812 3456 7890": "6281234567890",
		"81234567890":    "6281234567890",
		"":               "",
	}
	for in, want := range cases {
		if got := NormalizePhoneDigits(in); got != want {
			t.Errorf("%q => %q want %q", in, got, want)
		}
	}
}

func TestCatalogComplete(t *testing.T) {
	all := All()
	if len(all) != 26 {
		t.Fatalf("expected 26 templates, got %d", len(all))
	}
	seen := map[string]bool{}
	vendors, customers := 0, 0
	for _, tmpl := range all {
		if seen[tmpl.Code] {
			t.Fatalf("duplicate %s", tmpl.Code)
		}
		seen[tmpl.Code] = true
		if tmpl.Body == "" || len(tmpl.Variables) == 0 {
			t.Fatalf("%s incomplete", tmpl.Code)
		}
		switch tmpl.Audience {
		case AudienceVendor:
			vendors++
		case AudienceCustomer:
			customers++
		default:
			t.Fatalf("bad audience %s", tmpl.Code)
		}
	}
	if vendors != 8 || customers != 18 {
		t.Fatalf("vendors=%d customers=%d", vendors, customers)
	}
	for _, code := range []string{
		"WA-V01", "WA-V02", "WA-V03", "WA-V04", "WA-V05", "WA-V06", "WA-V07", "WA-V08",
		"WA-C01", "WA-C02", "WA-C03", "WA-C04", "WA-C05", "WA-C06", "WA-C07", "WA-C08",
		"WA-C09", "WA-C10", "WA-C11", "WA-C12", "WA-C13", "WA-C14", "WA-C15", "WA-C16",
		"WA-C17", "WA-C18",
	} {
		if !seen[code] {
			t.Fatalf("missing %s", code)
		}
	}
}

func TestMissingVariables(t *testing.T) {
	tmpl, ok := ByCode("WA-C12")
	if !ok {
		t.Fatal("missing WA-C12")
	}
	missing := MissingVariables(tmpl, map[string]string{
		"customer_name": "A", "order_number": "1", "store_name": "S",
		"carrier": "JNE", "tracking_number": "", "tracking_url": "http://x",
	})
	if len(missing) != 1 || missing[0] != "tracking_number" {
		t.Fatalf("missing=%v", missing)
	}
}
