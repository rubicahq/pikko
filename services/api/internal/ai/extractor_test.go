package ai

import "testing"

func TestNormalizePrices(t *testing.T) {
	legacy := int64(1000)
	e := ExtractedItem{PriceIDR: &legacy}
	e.NormalizePrices()
	if e.PriceNormalIDR == nil || *e.PriceNormalIDR != 1000 {
		t.Fatalf("expected legacy price_idr mapped to normal, got %#v", e.PriceNormalIDR)
	}

	normal, discount := int64(2000), int64(1500)
	e2 := ExtractedItem{PriceNormalIDR: &normal, PriceDiscountIDR: &discount}
	e2.NormalizePrices()
	if e2.PriceIDR == nil || *e2.PriceIDR != 1500 {
		t.Fatalf("expected effective price_idr = discount, got %#v", e2.PriceIDR)
	}

	e3 := ExtractedItem{}
	e3.NormalizePrices()
	if e3.PriceIDR != nil || e3.PriceNormalIDR != nil {
		t.Fatal("null prices must stay null")
	}
}
