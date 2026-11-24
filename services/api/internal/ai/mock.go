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
	price := int64(150000)
	sku := "MOCK-001"
	cat := "Apparel"
	page := 1
	x, y, w, h := 10.0, 20.0, 30.0, 25.0
	warn := "Price confidence medium"
	return &MockExtractor{
		Items: []ExtractedItem{
			{
				Name: &name, Brand: &brand, PriceIDR: &price, SKU: &sku, Category: &cat,
				PageNumber: &page, BBoxXPct: &x, BBoxYPct: &y, BBoxWidthPct: &w, BBoxHeightPct: &h,
				Uncertain: true, Warning: &warn, VariantInfo: json.RawMessage(`{"Size":["S","M","L"]}`),
			},
			{
				Name: ptr("Canvas Tote"), PriceIDR: ptrInt(89000), PageNumber: &page,
				Uncertain: false, VariantInfo: json.RawMessage(`{}`),
			},
		},
	}
}

func ptr(s string) *string { return &s }
func ptrInt(n int64) *int64 { return &n }

func (m *MockExtractor) ExtractCatalog(_ context.Context, req ExtractRequest) (*ExtractResult, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	items := m.Items
	if req.MaxItems > 0 && len(items) > req.MaxItems {
		items = items[:req.MaxItems]
	}
	// Attach page numbers from request when present
	if len(req.Pages) > 0 {
		for i := range items {
			if items[i].PageNumber == nil {
				p := req.Pages[0].PageNumber
				items[i].PageNumber = &p
			}
		}
	}
	raw, _ := json.Marshal(map[string]any{"items": items, "provider": "mock"})
	return &ExtractResult{Items: items, Raw: raw}, nil
}

// Ensure mock satisfies interface
var _ CatalogExtractor = (*MockExtractor)(nil)

func (m *MockExtractor) String() string {
	return fmt.Sprintf("MockExtractor(%d items)", len(m.Items))
}
