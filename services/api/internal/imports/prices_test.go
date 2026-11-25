package imports

import "testing"

func TestResolveImportPrices(t *testing.T) {
	normal, discount := int64(200000), int64(150000)
	base, compare, contact := resolveImportPrices(Item{
		PriceNormalIDR: &normal, PriceDiscountIDR: &discount,
	})
	if contact || base != 150000 || compare == nil || *compare != 200000 {
		t.Fatalf("dual price: base=%d compare=%v contact=%v", base, compare, contact)
	}

	base, compare, contact = resolveImportPrices(Item{PriceNormalIDR: &normal})
	if contact || base != 200000 || compare != nil {
		t.Fatalf("normal only: base=%d compare=%v contact=%v", base, compare, contact)
	}

	legacy := int64(99000)
	base, compare, contact = resolveImportPrices(Item{PriceIDR: &legacy})
	if contact || base != 99000 || compare != nil {
		t.Fatalf("legacy: base=%d compare=%v contact=%v", base, compare, contact)
	}

	base, compare, contact = resolveImportPrices(Item{})
	if !contact || base != 0 || compare != nil {
		t.Fatalf("null: base=%d compare=%v contact=%v", base, compare, contact)
	}
}
