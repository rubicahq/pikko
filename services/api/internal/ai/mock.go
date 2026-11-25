package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// MockExtractor returns deterministic items for tests/CI — never calls a network AI.
type MockExtractor struct {
	Items []ExtractedItem
	Err   error
}

func NewMockExtractor() *MockExtractor {
	name := "Sample Tee"
	brand := "MockBrand"
	normal := int64(175000)
	discount := int64(150000)
	sku := "MOCK-001"
	cat := "Apparel"
	page := 1
	x, y, w, h := 10.0, 20.0, 30.0, 25.0
	warn := "Price confidence medium"
	confLow, confHigh := 58.0, 92.0
	noPriceName := "Ask Seller Scarf"
	page2 := 2
	return &MockExtractor{
		Items: []ExtractedItem{
			{
				Name: &name, Brand: &brand, PriceNormalIDR: &normal, PriceDiscountIDR: &discount,
				PriceIDR: &discount, SKU: &sku, Category: &cat,
				PageNumber: &page, BBoxXPct: &x, BBoxYPct: &y, BBoxWidthPct: &w, BBoxHeightPct: &h,
				ConfidencePct: &confLow, Uncertain: true, Warning: &warn,
				VariantInfo: json.RawMessage(`{"Size":["S","M","L"]}`),
			},
			{
				Name: ptr("Canvas Tote"), PriceNormalIDR: ptrInt(89000), PriceIDR: ptrInt(89000),
				PageNumber: &page, ConfidencePct: &confHigh, Uncertain: false,
				VariantInfo: json.RawMessage(`{}`),
			},
			{
				Name: &noPriceName, PageNumber: &page2, ConfidencePct: &confHigh, Uncertain: false,
				VariantInfo: json.RawMessage(`{}`),
				Warning:     ptr("No price printed on page"),
			},
		},
	}
}

func ptr(s string) *string     { return &s }
func ptrInt(n int64) *int64    { return &n }

func (m *MockExtractor) ExtractCatalog(_ context.Context, req ExtractRequest) (*ExtractResult, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	items := append([]ExtractedItem(nil), m.Items...)
	if len(req.Pages) == 1 {
		// Per-page extraction: only return items for that page (or unpaged).
		page := req.Pages[0].PageNumber
		filtered := items[:0]
		for _, it := range items {
			if it.PageNumber == nil || *it.PageNumber == page {
				cp := it
				p := page
				cp.PageNumber = &p
				cp.NormalizePrices()
				cp.NormalizeConfidence()
				filtered = append(filtered, cp)
			}
		}
		items = filtered
	} else if len(req.Pages) > 0 {
		for i := range items {
			if items[i].PageNumber == nil {
				p := req.Pages[0].PageNumber
				items[i].PageNumber = &p
			}
			items[i].NormalizePrices()
			items[i].NormalizeConfidence()
		}
	} else {
		for i := range items {
			items[i].NormalizePrices()
			items[i].NormalizeConfidence()
		}
	}
	if req.MaxItems > 0 && len(items) > req.MaxItems {
		items = items[:req.MaxItems]
	}
	raw, _ := json.Marshal(map[string]any{"items": items, "provider": "mock"})
	return &ExtractResult{Items: items, Raw: raw}, nil
}

// Ensure mock satisfies interface
var _ CatalogExtractor = (*MockExtractor)(nil)

func (m *MockExtractor) String() string {
	return fmt.Sprintf("MockExtractor(%d items)", len(m.Items))
}
