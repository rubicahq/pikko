package imports

import (
	"strings"
	"testing"
)

func TestParseProductCSV(t *testing.T) {
	rows, err := parseProductCSV([]byte(CSVTemplate), 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0].Name == nil || *rows[0].Name != "Sample Tee" {
		t.Fatalf("name=%v", rows[0].Name)
	}
	if rows[0].PriceIDR == nil || *rows[0].PriceIDR != 125000 {
		t.Fatalf("price=%v", rows[0].PriceIDR)
	}
	if !rows[0].ReadyStock || rows[0].Preorder {
		t.Fatalf("eligibility ready=%v preorder=%v", rows[0].ReadyStock, rows[0].Preorder)
	}
	if rows[1].ReadyStock || !rows[1].Preorder {
		t.Fatalf("row2 eligibility ready=%v preorder=%v", rows[1].ReadyStock, rows[1].Preorder)
	}
}

func TestParseProductCSVMissingColumns(t *testing.T) {
	_, err := parseProductCSV([]byte("foo,bar\n1,2\n"), 500)
	if err == nil || !strings.Contains(err.Error(), "missing required column") {
		t.Fatalf("expected missing column error, got %v", err)
	}
}

func TestParseProductCSVDuplicateSKU(t *testing.T) {
	raw := "name,price_idr,sku\nA,1000,SKU1\nB,2000,SKU1\n"
	rows, err := parseProductCSV([]byte(raw), 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[1].Warning == nil || !strings.Contains(*rows[1].Warning, "duplicate SKU") {
		t.Fatalf("warning=%v", rows[1].Warning)
	}
}
