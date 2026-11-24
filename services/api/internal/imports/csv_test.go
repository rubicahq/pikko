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
	if rows[0].Confidence == nil || *rows[0].Confidence < 80 {
		t.Fatalf("expected high confidence, got %v", rows[0].Confidence)
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
	if err == nil {
		t.Fatal("expected warning summary error")
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[1].Warning == nil || !strings.Contains(*rows[1].Warning, "duplicate SKU") {
		t.Fatalf("warning=%v", rows[1].Warning)
	}
}

func TestParseProductCSVDecimalPrice(t *testing.T) {
	raw := "name,price_idr\nTee,125000.50\n"
	rows, err := parseProductCSV([]byte(raw), 500)
	if err == nil {
		t.Fatal("expected warning error")
	}
	if len(rows) != 1 || rows[0].Selected {
		t.Fatalf("expected rejected row, got %+v", rows)
	}
	if rows[0].Warning == nil || !strings.Contains(*rows[0].Warning, "whole rupiah") {
		t.Fatalf("warning=%v", rows[0].Warning)
	}
}

func TestParseProductCSVSemicolonRejected(t *testing.T) {
	raw := "name;price_idr\nTee;1000\n"
	_, err := parseProductCSV([]byte(raw), 500)
	if err == nil || !strings.Contains(err.Error(), "comma") {
		t.Fatalf("expected comma separator error, got %v", err)
	}
}

func TestParseProductCSVEmpty(t *testing.T) {
	_, err := parseProductCSV([]byte("name,price_idr\n\n"), 500)
	if err == nil || !strings.Contains(err.Error(), "no data rows") {
		t.Fatalf("expected no data rows, got %v", err)
	}
}

func TestParseProductCSVBOMAndIDRPrefix(t *testing.T) {
	raw := "\xef\xbb\xbfname,price_idr\nBag,Rp 99000\n"
	// BOM stripped by StartCSV; parser receives cleaned bytes in production —
	// here we only check IDR/Rp prefix handling after BOM strip.
	raw = strings.TrimPrefix(raw, "\xef\xbb\xbf")
	rows, err := parseProductCSV([]byte(raw), 500)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].PriceIDR == nil || *rows[0].PriceIDR != 99000 {
		t.Fatalf("price=%v", rows[0].PriceIDR)
	}
}

func TestParseProductCSVInvalidBool(t *testing.T) {
	raw := "name,price_idr,ready_stock_eligible\nBag,1000,maybe\n"
	rows, err := parseProductCSV([]byte(raw), 500)
	if err == nil {
		t.Fatal("expected warning")
	}
	if rows[0].Warning == nil || !strings.Contains(*rows[0].Warning, "ready_stock_eligible") {
		t.Fatalf("warning=%v", rows[0].Warning)
	}
}

func TestClampHotspotBBox(t *testing.T) {
	x, y, w, h, ok := clampHotspotBBox(-5, 10, 30, 20)
	if !ok || x != 0 || y != 10 || w != 30 || h != 20 {
		t.Fatalf("got %v %v %v %v ok=%v", x, y, w, h, ok)
	}
	_, _, _, _, ok = clampHotspotBBox(95, 95, 20, 20)
	if !ok {
		t.Fatal("expected clamped box ok")
	}
	_, _, _, _, ok = clampHotspotBBox(0, 0, 0, 10)
	if ok {
		t.Fatal("zero width should fail")
	}
}
