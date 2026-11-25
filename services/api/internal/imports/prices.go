package imports

// resolveImportPrices maps catalog-item dual/null prices onto product fields.
// discount → base; normal (when both) → compare_at; both null → contact_for_price.
func resolveImportPrices(it Item) (base int64, compareAt *int64, contact bool) {
	normal := it.PriceNormalIDR
	discount := it.PriceDiscountIDR
	if normal == nil && discount == nil && it.PriceIDR != nil {
		normal = it.PriceIDR
	}
	if discount != nil {
		base = *discount
		if normal != nil && *normal > *discount {
			compareAt = normal
		}
		return base, compareAt, false
	}
	if normal != nil {
		return *normal, nil, false
	}
	return 0, nil, true
}

// effectivePriceIDR mirrors AI NormalizePrices for persistence on price_idr.
func effectivePriceIDR(normal, discount *int64) *int64 {
	if discount != nil {
		return discount
	}
	if normal != nil {
		return normal
	}
	return nil
}
